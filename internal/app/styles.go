package app

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

type appStyles struct {
	statusBar          lipgloss.Style
	errorText          lipgloss.Style
	editorBox          lipgloss.Style
	hintHeader         lipgloss.Style
	hintRow            lipgloss.Style
	recordHeader       lipgloss.Style
	recordField        lipgloss.Style
	recordSep          lipgloss.Style
	recordCursorLine   lipgloss.Style
	recordSelectedMark lipgloss.Style
	tableStyle         table.Styles
}

func newStyles() *appStyles {
	base := table.DefaultStyles()
	base.Header = base.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	base.Selected = base.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	return &appStyles{
		statusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1),
		errorText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true),
		editorBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1),
		recordHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(0, 0, 0, 2),
		recordField: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true),
		recordSep: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		recordCursorLine: lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true),
		recordSelectedMark: lipgloss.NewStyle().
			Foreground(lipgloss.Color("82")).
			Bold(true),
		hintHeader: lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")).
			Bold(true).
			Padding(1, 0, 0, 2),
		hintRow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			PaddingLeft(2),
		tableStyle: base,
	}
}
