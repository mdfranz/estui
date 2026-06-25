package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/mdfranz/estui/internal/messages"
)

type appState int

const (
	stateConnecting appState = iota
	stateReady
	stateRunning
	stateError
)

type viewMode int

const (
	viewModeTable viewMode = iota
	viewModeRecord
)

const editorHeight = 8
const recordHeaderLines = 2 // header + blank line before fields

type recordField struct {
	name string
	val  string
}

type Model struct {
	client         *elasticsearch.TypedClient
	state          appState
	editor         textarea.Model
	table          table.Model
	record         viewport.Model
	spinner        spinner.Model
	styles         *appStyles
	err            error
	cancelFn       context.CancelFunc
	hints          []messages.IndexHint
	viewMode       viewMode
	allColumns     []string
	allRows        [][]string
	recordFields   []recordField
	recordCursor   int
	selectedFields map[string]bool
	rowCount       int
	width          int
	height         int
}

func New(client *elasticsearch.TypedClient) *Model {
	ta := textarea.New()
	ta.Placeholder = "Enter ES|QL query… (Ctrl+R to run, Esc to cancel, Tab to switch focus)"
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(editorHeight)
	ta.ShowLineNumbers = true

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	st := newStyles()

	tbl := table.New(table.WithFocused(false))
	tbl.SetStyles(st.tableStyle)

	vp := viewport.New(80, 20)

	return &Model{
		client:         client,
		state:          stateConnecting,
		editor:         ta,
		table:          tbl,
		record:         vp,
		spinner:        sp,
		styles:         st,
		selectedFields: make(map[string]bool),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		ping(context.Background(), m.client),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeComponents()
		return m, nil

	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC:
			return m, tea.Quit

		case msg.String() == "q" && m.state == stateReady && !m.editor.Focused() && m.viewMode == viewModeTable:
			return m, tea.Quit

		case msg.Type == tea.KeyEsc && m.state == stateRunning:
			if m.cancelFn != nil {
				m.cancelFn()
			}
			m.state = stateReady
			return m, nil

		// --- record view navigation ---

		case (msg.Type == tea.KeyEsc || msg.String() == "v") && m.viewMode == viewModeRecord:
			m.viewMode = viewModeTable
			return m, nil

		case (msg.String() == "j" || msg.Type == tea.KeyDown) && m.viewMode == viewModeRecord:
			m.moveCursor(1)
			return m, nil

		case (msg.String() == "k" || msg.Type == tea.KeyUp) && m.viewMode == viewModeRecord:
			m.moveCursor(-1)
			return m, nil

		case msg.String() == " " && m.viewMode == viewModeRecord:
			m.toggleFieldSelection()
			return m, nil

		// Enter in record view: append | WHERE field == value, return to editor
		case msg.Type == tea.KeyEnter && m.viewMode == viewModeRecord:
			if m.recordCursor < len(m.recordFields) {
				f := m.recordFields[m.recordCursor]
				m.appendToQuery(formatWhereClause(f.name, f.val))
			}
			m.viewMode = viewModeTable
			m.table.Blur()
			m.editor.Focus()
			return m, nil

		// Ctrl+K in record view: append | KEEP <selected fields>, return to editor
		case msg.Type == tea.KeyCtrlK && m.viewMode == viewModeRecord:
			m.appendKeep()
			m.viewMode = viewModeTable
			m.table.Blur()
			m.editor.Focus()
			return m, nil

		case (msg.String() == "n" || msg.String() == "]") && m.viewMode == viewModeRecord:
			m.table.MoveDown(1)
			m.recordCursor = 0
			m.record.SetContent(m.buildRecordContent())
			m.record.GotoTop()
			return m, nil

		case (msg.String() == "p" || msg.String() == "[") && m.viewMode == viewModeRecord:
			m.table.MoveUp(1)
			m.recordCursor = 0
			m.record.SetContent(m.buildRecordContent())
			m.record.GotoTop()
			return m, nil

		// --- table / editor navigation ---

		case (msg.String() == "v" || msg.Type == tea.KeyEnter) && !m.editor.Focused() && m.rowCount > 0 && m.viewMode == viewModeTable:
			m.recordCursor = 0
			m.viewMode = viewModeRecord
			m.record.SetContent(m.buildRecordContent())
			m.record.GotoTop()
			return m, nil

		case msg.Type == tea.KeyTab:
			if m.viewMode == viewModeRecord {
				m.viewMode = viewModeTable
				m.table.Blur()
				m.editor.Focus()
			} else if m.editor.Focused() {
				m.editor.Blur()
				m.table.Focus()
			} else {
				m.table.Blur()
				m.editor.Focus()
			}
			return m, nil

		case msg.Type == tea.KeyCtrlR || (msg.Type == tea.KeyEnter && msg.Alt):
			if m.state == stateReady {
				return m, m.startQuery()
			}

		default:
			if m.state == stateReady && m.editor.Focused() && strings.TrimSpace(m.editor.Value()) == "" {
				if idx := digitKey(msg.String()); idx >= 1 && idx <= len(m.hints) {
					q := hintQuery(m.hints[idx-1].Index)
					m.editor.SetValue(q)
					m.editor.CursorEnd()
					return m, nil
				}
			}
		}

	case spinner.TickMsg:
		if m.state == stateConnecting || m.state == stateRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case messages.PingResultMsg:
		if msg.Err != nil {
			m.state = stateError
			m.err = fmt.Errorf("connection failed: %w", msg.Err)
			return m, nil
		}
		m.state = stateReady
		return m, loadHints(context.Background(), m.client)

	case messages.HintsMsg:
		if msg.Err == nil {
			m.hints = msg.Hints
		}
		return m, nil

	case messages.QueryResultMsg:
		m.state = stateReady
		m.viewMode = viewModeTable
		m.cancelFn = nil
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.err = nil
			m.rowCount = len(msg.Rows)
			m.applyResults(msg.Columns, msg.Rows)
		}
		return m, nil
	}

	var cmds []tea.Cmd
	if m.editor.Focused() {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.viewMode == viewModeRecord {
		// only pass page-up/down through to the viewport; cursor keys handled above
		var cmd tea.Cmd
		m.record, cmd = m.record.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

// --- record view helpers ---

func (m *Model) moveCursor(delta int) {
	if len(m.recordFields) == 0 {
		return
	}
	m.recordCursor += delta
	if m.recordCursor < 0 {
		m.recordCursor = 0
	}
	if m.recordCursor >= len(m.recordFields) {
		m.recordCursor = len(m.recordFields) - 1
	}
	m.record.SetContent(m.buildRecordContent())
	m.scrollToCursor()
}

func (m *Model) scrollToCursor() {
	line := recordHeaderLines + m.recordCursor
	if line < m.record.YOffset {
		m.record.SetYOffset(line)
	} else if line >= m.record.YOffset+m.record.Height {
		m.record.SetYOffset(line - m.record.Height + 1)
	}
}

func (m *Model) toggleFieldSelection() {
	if m.recordCursor >= len(m.recordFields) {
		return
	}
	name := m.recordFields[m.recordCursor].name
	if m.selectedFields[name] {
		delete(m.selectedFields, name)
	} else {
		m.selectedFields[name] = true
	}
	m.record.SetContent(m.buildRecordContent())
}

func (m *Model) appendKeep() {
	var fields []string
	if len(m.selectedFields) > 0 {
		for _, f := range m.recordFields {
			if m.selectedFields[f.name] {
				fields = append(fields, f.name)
			}
		}
	} else {
		for _, f := range m.recordFields {
			fields = append(fields, f.name)
		}
	}
	if len(fields) > 0 {
		m.appendToQuery("| KEEP " + strings.Join(fields, ", "))
	}
}

func (m *Model) appendToQuery(clause string) {
	current := strings.TrimRight(m.editor.Value(), " \n\t")
	m.editor.SetValue(current + "\n" + clause)
	m.editor.CursorEnd()
}

func formatWhereClause(field, val string) string {
	if _, err := strconv.ParseFloat(val, 64); err == nil {
		return fmt.Sprintf("| WHERE %s == %s", field, val)
	}
	return fmt.Sprintf("| WHERE %s == \"%s\"", field, val)
}

// --- query execution ---

func (m *Model) startQuery() tea.Cmd {
	q := strings.TrimSpace(m.editor.Value())
	if q == "" {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFn = cancel
	m.state = stateRunning
	return tea.Batch(
		m.spinner.Tick,
		runQuery(ctx, m.client, q),
	)
}

func (m *Model) applyResults(cols []string, rows [][]string) {
	if len(cols) == 0 {
		return
	}

	m.allColumns = cols
	m.allRows = rows
	m.selectedFields = make(map[string]bool)
	m.recordCursor = 0

	keep := make([]bool, len(cols))
	for _, row := range rows {
		for i, v := range row {
			if i < len(keep) && v != "" && v != "null" {
				keep[i] = true
			}
		}
	}

	var activeCols []string
	for i, c := range cols {
		if keep[i] {
			activeCols = append(activeCols, c)
		}
	}
	if len(activeCols) == 0 {
		activeCols = cols
	}

	colWidth := (m.width - len(activeCols)) / len(activeCols)
	if colWidth < 8 {
		colWidth = 8
	}
	if colWidth > 40 {
		colWidth = 40
	}

	tableCols := make([]table.Column, len(activeCols))
	for i, c := range activeCols {
		w := colWidth
		if len(c) > w {
			w = len(c) + 2
		}
		tableCols[i] = table.Column{Title: c, Width: w}
	}

	activeIdx := make([]int, 0, len(activeCols))
	for i := range cols {
		if keep[i] {
			activeIdx = append(activeIdx, i)
		}
	}

	tableRows := make([]table.Row, len(rows))
	for i, row := range rows {
		cells := make([]string, len(activeIdx))
		for j, idx := range activeIdx {
			if idx < len(row) {
				cells[j] = row[idx]
			}
		}
		tableRows[i] = table.Row(cells)
	}

	m.table.SetColumns(tableCols)
	m.table.SetRows(tableRows)
	m.table.GotoTop()
}

// --- record content builder ---

func (m *Model) buildRecordContent() string {
	cursor := m.table.Cursor()
	if len(m.allRows) == 0 || cursor >= len(m.allRows) {
		m.recordFields = nil
		return "No row selected."
	}
	row := m.allRows[cursor]

	var fields []recordField
	for i, c := range m.allColumns {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		if val == "" || val == "null" {
			continue
		}
		fields = append(fields, recordField{c, val})
	}
	m.recordFields = fields

	if m.recordCursor >= len(fields) {
		m.recordCursor = max(0, len(fields)-1)
	}

	nameWidth := 0
	for _, f := range fields {
		if len(f.name) > nameWidth {
			nameWidth = len(f.name)
		}
	}

	selCount := len(m.selectedFields)
	header := m.styles.recordHeader.Render(fmt.Sprintf("Row %d of %d", cursor+1, m.rowCount))
	sub := m.styles.recordSep.Render(fmt.Sprintf("  (%d of %d fields set, %d selected)\n\n", len(fields), len(m.allColumns), selCount))

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(sub)

	for i, f := range fields {
		selected := m.selectedFields[f.name]
		if i == m.recordCursor {
			mark := "  "
			if selected {
				mark = "✓ "
			}
			line := mark + fmt.Sprintf("%-*s", nameWidth, f.name) + " │ " + f.val
			sb.WriteString(m.styles.recordCursorLine.Render(line) + "\n")
		} else {
			mark := "  "
			if selected {
				mark = m.styles.recordSelectedMark.Render("✓ ")
			}
			fieldName := m.styles.recordField.Render(fmt.Sprintf("%-*s", nameWidth, f.name))
			sep := m.styles.recordSep.Render(" │ ")
			sb.WriteString(mark + fieldName + sep + f.val + "\n")
		}
	}
	return sb.String()
}

// --- layout ---

func (m *Model) resizeComponents() {
	m.editor.SetWidth(m.width - m.styles.editorBox.GetHorizontalFrameSize())

	bodyHeight := m.height - editorHeight - m.styles.editorBox.GetVerticalFrameSize() - 2
	if bodyHeight < 3 {
		bodyHeight = 3
	}
	m.table.SetWidth(m.width)
	m.table.SetHeight(bodyHeight)
	m.record.Width = m.width
	m.record.Height = bodyHeight
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	editorView := m.styles.editorBox.Render(m.editor.View())
	status := m.statusLine()

	var body string
	switch m.state {
	case stateConnecting:
		body = fmt.Sprintf("\n  %s Connecting to Elasticsearch…", m.spinner.View())
	case stateRunning:
		body = fmt.Sprintf("\n  %s Running query… (Esc to cancel)", m.spinner.View())
	case stateError:
		body = "\n  " + m.styles.errorText.Render("Error: "+m.err.Error())
	default:
		if m.err != nil {
			body = "\n  " + m.styles.errorText.Render(m.err.Error()) + "\n" + m.table.View()
		} else if m.rowCount == 0 && len(m.hints) > 0 {
			body = m.hintsView()
		} else if m.viewMode == viewModeRecord {
			body = m.record.View()
		} else {
			body = m.table.View()
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, editorView, status, body)
}

func (m *Model) hintsView() string {
	var sb strings.Builder
	sb.WriteString(m.styles.hintHeader.Render("Active indices — last 15 minutes (press 1–9 to load)"))
	sb.WriteByte('\n')
	for i, h := range m.hints {
		label := fmt.Sprintf("  [%d] %-50s %s events", i+1, h.Index, h.Count)
		sb.WriteString(m.styles.hintRow.Render(label))
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (m *Model) statusLine() string {
	var parts []string
	switch m.state {
	case stateConnecting:
		parts = append(parts, "connecting…")
	case stateRunning:
		parts = append(parts, "running")
	case stateError:
		parts = append(parts, "error")
	default:
		parts = append(parts, "ready")
		if m.rowCount > 0 {
			activeCols := len(m.table.Columns())
			totalCols := len(m.allColumns)
			colInfo := fmt.Sprintf("%d rows  %d cols", m.rowCount, activeCols)
			if totalCols > activeCols {
				colInfo += fmt.Sprintf(" (%d null-only hidden)", totalCols-activeCols)
			}
			parts = append(parts, colInfo)
		}
	}

	if m.viewMode == viewModeRecord {
		selCount := len(m.selectedFields)
		hint := "j/k: move  Space: select"
		if selCount > 0 {
			hint += fmt.Sprintf(" (%d)", selCount)
		}
		hint += "  Enter: WHERE  Ctrl+K: KEEP  n/p: next/prev  Esc: back"
		parts = append(parts, hint)
	} else {
		parts = append(parts, "Ctrl+R: run  Tab: focus  Enter/v: record view  Ctrl+C: quit")
	}

	return m.styles.statusBar.Width(m.width).Render(strings.Join(parts, "  │  "))
}

// --- small helpers ---

func hintQuery(index string) string {
	return fmt.Sprintf("FROM %s\n| WHERE @timestamp >= NOW() - 15 minutes\n| LIMIT 100", index)
}

func digitKey(s string) int {
	if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
		return int(s[0] - '0')
	}
	return 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
