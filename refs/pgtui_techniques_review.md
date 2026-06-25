# PGTUI Codebase Review: Architecture & Reusable TUI Techniques

This document reviews the architectural design, patterns, and terminal UI (TUI) techniques used in the [pgtui](file:///home/mdfranz/github/pgtui) codebase. The goal is to extract these techniques so they can be adapted to build a terminal client for a different query engine (e.g., SQLite, MySQL, ClickHouse, DuckDB, or Trino).

---

## 1. Architectural Patterns & Decoupling

Building interactive terminal applications with Charm's **Bubble Tea** (Elm Architecture) often leads to monolithic file creep because all states, keyboard actions, and side effects flow through a single `Update(msg tea.Msg) (tea.Model, tea.Cmd)` loop. 

`pgtui` resolves this by using two key decoupling patterns:

### The Delegate/Controller Pattern
Instead of putting all message routing in `internal/app/app.go`, the application defines a `Delegate` interface:
```go
type Delegate interface {
    Name() string
    Update(msg tea.Msg, app AppAccess) (handled bool, cmd tea.Cmd)
}
```
- **`AppAccess` Interface**: Controls what delegates can do (e.g., trigger queries, retrieve components, change focus area). It isolates the delegates from direct struct manipulation of the main application.
- **Domain Delegates**:
  - [`TreeDelegate`](file:///home/mdfranz/github/pgtui/internal/app/delegates/tree.go): Handles navigation tree selections, schema browsing, and lazy loading node children.
  - [`DataDelegate`](file:///home/mdfranz/github/pgtui/internal/app/delegates/data.go): Coordinates populating table views, pagination, prefetching, and metadata changes.
  - [`QueryDelegate`](file:///home/mdfranz/github/pgtui/internal/app/delegates/query.go): Manages raw SQL execution, tab creation, and query cancellation.
  - [`ConnectionDelegate`](file:///home/mdfranz/github/pgtui/internal/app/delegates/connection.go): Manages connection profiles, manual entry forms, and background host discovery.

### Messaging Package
To prevent Go import cycles between the delegates package, UI components, and the main app, all asynchronous event types are declared in a standalone [`internal/app/messages/messages.go`](file:///home/mdfranz/github/pgtui/internal/app/messages/messages.go) package. This ensures any component can emit commands without needing references to the app controller.

### Compile-Time Interface Assertion
`app_access.go` enforces the `AppAccess` contract at compile time rather than at runtime:
```go
var _ delegates.AppAccess = (*App)(nil)
```
If `App` ever drifts from the interface, the build fails immediately. Use this pattern whenever you implement a non-trivial interface to catch drift early.

---

## 2. Asynchronous Execution & Query Management

TUIs run on a single-threaded rendering loop. Blocking this loop for database queries or metadata scanning freezes the terminal, causing a terrible user experience.

### Non-blocking Background Commands (`tea.Cmd`)
Every database task (like querying table data, loading schema, or running a query) is wrapped in a `tea.Cmd`, which spawns a goroutine:
```go
func (a *App) loadTableData(msg messages.LoadTableDataMsg) tea.Cmd {
    return func() tea.Msg {
        ctx := context.Background()
        conn, err := a.connectionManager.GetActive()
        if err != nil {
            return messages.TableDataLoadedMsg{Err: err}
        }
        data, err := metadata.QueryTableData(ctx, conn.Pool, msg.Schema, msg.Table, msg.Offset, msg.Limit, sort)
        if err != nil {
            return messages.TableDataLoadedMsg{Err: err}
        }
        return messages.TableDataLoadedMsg{
            Columns: data.Columns,
            Rows:    data.Rows,
            Offset:  msg.Offset,
        }
    }
}
```

### Cancellable Queries
Long-running queries can hang or block connections. `pgtui` implements query cancellation using Contexts:
1. When starting a query, it creates a cancellable context:
   ```go
   ctx, cancel := context.WithCancel(context.Background())
   a.executeCancelFn = cancel
   ```
2. The cancel function is stored in the application struct.
3. If the user presses `Esc` while the loading spinner is active, the app invokes `a.executeCancelFn()`, interrupting the network request in the background, and restoring focus immediately.

---

## 3. High-Performance Tabular Grid (`TableView`)

Displaying database tables in a terminal requires handling wide columns, horizontal scrolling, and millions of rows efficiently. The implementation in [`internal/ui/components/table_view.go`](file:///home/mdfranz/github/pgtui/internal/ui/components/table_view.go) uses several key optimization techniques:

### Virtual Scrolling (Vertical and Horizontal)
Instead of rendering all rows and columns, `TableView` calculates what can fit inside the viewport dimensions (`VisibleRows` and `VisibleCols`):
- **Vertical**: Only loop and draw rows from index `TopRow` to `TopRow + VisibleRows`.
- **Horizontal**: Render cells starting from `LeftColOffset` up to the number of columns that fit within the remaining terminal width.
- **Bounds Auto-adjustment**: Whenever the selection cursor moves (`MoveSelection`), the component automatically computes if `TopRow` or `LeftColOffset` needs to shift to keep the cursor visible.

### Performance-Sensitive Width Calculations
Determining column widths requires scanning cell contents. Doing this on every tick or over a million rows is highly expensive.
- **Sampling**: `calculateColumnWidths()` only reads the first `100` rows of the active dataset.
- **Early Truncation**: Strings are truncated *before* measuring their width:
  ```go
  maxCheckLen := maxWidth * 4 // Capped column width threshold
  if len(cell) > maxCheckLen {
      cell = cell[:maxCheckLen]
  }
  cellLen := runewidth.StringWidth(checkCell)
  ```
  This prevents `runewidth.StringWidth()` (which handles multi-byte UTF-8 string measurements) from wasting CPU cycles scanning megabyte-sized TEXT/JSONB columns.

### Foreground Prefetching
For large datasets, scrolling to the bottom triggers a background prefetch command:
```go
func (tv *TableView) NeedsPrefetch() bool {
    remaining := len(tv.Rows) - tv.SelectedRow
    return remaining < tv.PrefetchThreshold && len(tv.Rows) < tv.TotalRows
}
```
If `NeedsPrefetch()` returns true, it triggers a non-blocking load for the *next* chunk of data, appending it to the in-memory array (`tv.Rows`). This achieves seamless scrolling without blocking the UI.

### Lip Gloss Style Caching
Creating `lipgloss.Style` objects inline inside a render loop produces heavy garbage collection overhead. `TableView` initializes a cache struct (`tableViewStyles`) once and reuses the styled instances:
```go
type MyComponent struct {
    cachedStyles *myStyles
}

func (c *MyComponent) initStyles() {
    c.cachedStyles = &myStyles{
        border: lipgloss.NewStyle().Foreground(theme.Border),
        // ...
    }
}
```

### Border-Aware Width Calculation
When a `lipgloss` style has a border or padding, `Width()` sets the content area width, but the border/padding renders *outside* that — causing overflow. Always use `GetHorizontalFrameSize()` to account for the frame:
```go
containerStyle := lipgloss.NewStyle().
    Border(lipgloss.RoundedBorder()).
    Padding(1, 2)

contentWidth := terminalWidth - containerStyle.GetHorizontalFrameSize()
content := renderContent(contentWidth)
return containerStyle.Render(content)
```
For nested components, each child subtracts its own frame size from the width passed down by its parent.

### LRU Cache for Rendered Rows
[`internal/ui/components/lru_cache.go`](file:///home/mdfranz/github/pgtui/internal/ui/components/lru_cache.go) provides a thread-safe `LRUCache` keyed by row index. Rather than re-processing and re-styling unchanged rows on every frame, the cache returns the pre-rendered `[]string` directly. The cache is cleared whenever columns or sort order changes. For large result sets, this is a meaningful render-path optimization.

### Mouse Support via bubblezone
Clickable regions are marked and detected using [bubblezone](https://github.com/lrstanley/bubblezone):
```go
// In View() — wrap the rendered content
zone.Mark("button-id", buttonContent)

// In Update() — check bounds on mouse events
if zone.Get("button-id").InBounds(mouseMsg) {
    // handle click
}
```
`zone.NewGlobal()` must be called in `main()` and also in the `init()` of any test file that calls `View()` methods, otherwise zone lookups panic.

---

## 4. Custom SQL Editor Component

Instead of binding external text area libraries, `pgtui` builds a customized multiline editor in [`internal/ui/components/sql_editor.go`](file:///home/mdfranz/github/pgtui/internal/ui/components/sql_editor.go):

### Line Buffer & Manual Cursor
- The editor holds code as `lines []string` and manages manual offsets for `cursorRow` and `cursorCol`.
- Keyboard bindings (left, right, up, down, backspace, delete, enter, tab) are processed by modifying the slice indices and splicing strings.

### Quote-Aware Statement Selection
In professional editors, executing a query with multiple statements executes the one under the cursor:
1. `splitStatements()` traverses the buffer, ignoring semicolons (`;`) when inside single-quoted strings (`'...'`).
2. It calculates the cumulative character offset of the cursor.
3. It finds which split statement contains the cursor character index and executes *only* that subset of the text.

### Interactive External Editor Integration
If a user wants to edit complex SQL in their preferred system editor (Vim, Nano, VS Code):
1. Write the current editor text area content to a temp file (`pgtui-*.sql`).
2. Retrieve the `$EDITOR` environment variable.
3. Spawn a subprocess:
   ```go
   cmd := exec.Command(editor, tmpFile.Name())
   cmd.Stdin = os.Stdin
   cmd.Stdout = os.Stdout
   cmd.Stderr = os.Stderr
   cmd.Run()
   ```
4. By piping standard streams directly, the external editor takes over the terminal temporarily.
5. Once the subprocess exits, read the temp file, update the TUI buffer content, and clean up the file.

---

## 5. Search, Commands, and Fuzzy Matching

### Fuzzy Search with Scoring
[`internal/search/fuzzy.go`](file:///home/mdfranz/github/pgtui/internal/search/fuzzy.go) implements a two-tier matching strategy used by the command palette and tree filter:
1. **Exact substring match**: scored 50–100 depending on how early in the target string the match appears.
2. **Character-by-character sequential match**: characters from the query must appear in order anywhere in the target. Score is boosted for characters that appear early or are consecutive.

```go
match := search.FuzzyMatch(query, target)
if match.Matched {
    // match.Score ranks results; match.Indices highlight matched chars
}
```
`RankMatches()` scores a slice of strings in one call. Adapt this for filtering any named entity: tables, collections, indexes, functions.

### Thread-Safe Command Registry
[`internal/commands/registry.go`](file:///home/mdfranz/github/pgtui/internal/commands/registry.go) stores commands in a `map[string]Command` protected by a `sync.RWMutex`. Each `Command` carries:
- `ID` — stable string key
- `Label` / `Description` — displayed in the palette
- `Tags []string` — searched alongside label/description

`Registry.Search()` does case-insensitive substring matching across all three fields. New engine-specific actions (e.g., `EXPLAIN`, `SHOW PROCESSLIST`, `DESCRIBE`) are registered at startup and appear in the palette automatically with no UI changes. `Unregister()` supports removing commands dynamically (e.g., when disconnected).

---

## 6. Collapsible Tree Viewer (JSONB Viewer)

Displaying hierarchical nested trees (like schema trees or JSON structures) recursively in Bubble Tea is slow and makes viewport scrolling extremely complex. 

`pgtui` solves this in [`internal/ui/components/jsonb_viewer.go`](file:///home/mdfranz/github/pgtui/internal/ui/components/jsonb_viewer.go) using a **Flattening Technique**:

### The Flat Array Mapping
1. Parse the JSON or schema into a recursive node graph (`TreeNode` struct containing `Parent`, `Children`, and `IsExpanded`).
2. Flatten the expanded elements into a single linear slice of pointers (`visibleNodes []*TreeNode`):
   ```go
   func (jv *JSONBViewer) flattenTree(node *TreeNode) {
       jv.visibleNodes = append(jv.visibleNodes, node)
       if node.IsExpanded && len(node.Children) > 0 {
           for _, child := range node.Children {
               jv.flattenTree(child)
           }
       }
   }
   ```
3. Navigation (arrow keys, mouse clicks, page scrolls) is performed directly against this simple `visibleNodes` slice.
4. When a node is toggled, its `IsExpanded` state changes, `rebuildVisibleNodes()` is called, and the selection index is clamped.

### Path Assembly
Because each `TreeNode` maintains its depth (`Level`) and a key slice path (`Path []string`), the viewer can reconstruct the path back to the root node. In a Postgres context, this outputs the exact path operators:
- Object key path: `-> 'config' ->> 'port'`
- Array key path: `-> 'items' -> 0`

Adapting this for another query engine is simple (e.g. producing standard JSONPath `$.config.port` for MySQL or ClickHouse).

---

## 7. Local Persistence

### Embedded SQLite Schema with `//go:embed`
[`internal/history/store.go`](file:///home/mdfranz/github/pgtui/internal/history/store.go) uses `database/sql` + `go-sqlite3` for query history, with the DDL embedded directly into the binary:
```go
//go:embed schema.sql
var schemaSQL string

func NewStore(path string) (*Store, error) {
    db, _ := sql.Open("sqlite3", path)
    db.Exec(schemaSQL) // idempotent CREATE TABLE IF NOT EXISTS
    return &Store{db: db}, nil
}
```
Each history entry records connection name, database, SQL, duration (ms), rows affected, success flag, and error message. `Store.Search()` does a `LIKE` query against the `query` column for history recall.

This pattern replaces ad-hoc JSON/flat-file approaches. A single SQLite file at `~/.config/<appname>/app.db` handles query history, favorites, recent connections, and user preferences — zero external dependencies, zero setup, travels with the binary.

### Layered Connection Discovery
[`internal/db/discovery/`](file:///home/mdfranz/github/pgtui/internal/db/discovery/) coordinates three discovery sources, run in priority order:
1. **Environment variables** (`PGHOST`, `DATABASE_URL`, etc.)
2. **Local port scan** — concurrent TCP dial against the default port on localhost
3. **Config files** — parse `~/.pgpass` or engine-specific config

Results are deduplicated by `host:port` key, keeping the highest-priority source:
```go
key := instance.Host + ":" + strconv.Itoa(instance.Port)
if existing, exists := seen[key]; !exists || instance.Source < existing.Source {
    seen[key] = instance
}
```
For a new engine, substitute the relevant env var names, port numbers, and config file paths. The deduplication and priority logic is unchanged.

### Export Formats
[`internal/export/exporter.go`](file:///home/mdfranz/github/pgtui/internal/export/exporter.go) provides `ExportToCSV()` and `ExportToJSON()` that operate on `[]models.Favorite` (or any `[][]string` result set). Because query results are already normalized to `[]string` rows by `executor.go`, adding export to any result view is straightforward. Both functions write to a caller-supplied file path and return a typed error.

---

## 8. How to Port these Techniques to a New Query Engine

If you are writing a query tool for a different engine (e.g., ClickHouse, DuckDB, Trino):

| Layer | What to Replace | What Stays Intact |
|---|---|---|
| `internal/db/metadata/` | Catalog queries (`pg_tables` → `information_schema`, `system.tables`, `sqlite_master`) | Message types, `TreeNode` model |
| `internal/db/query/executor.go` | Driver import + `convertValueToString()` type switch | Non-blocking `tea.Cmd` wrapper, `QueryResult` shape |
| `internal/db/connection/` | pgx pool → `database/sql` + target driver | Connection manager interface |
| `internal/db/discovery/` | Env var names, port numbers, config file paths | Layered discovery + deduplication logic |
| `internal/models/` | Add engine-specific `TreeNodeType` constants | Generic `QueryResult`, `TreeNode`, `Filter` |
| `internal/ui/` | Nothing | All components — table, tree, editor, JSONB, command palette |
| `internal/history/` | Nothing (SQLite is generic) | Query history store |
| `internal/search/` | Nothing | Fuzzy matcher |
| `internal/commands/` | Register engine-specific commands | Registry, palette, search |

**Key invariant**: UI components consume `models.QueryResult{Columns []string, Rows [][]string}`. Any engine that produces that struct works with the existing table view, virtual scrolling, prefetch, LRU cache, search, and export layers unchanged.

1. **Keep the UI Layer Intact**: The `TableView`, `TreeView`, `SQLEditor`, and `JSONBViewer` components are highly decoupled. They take Go models (`models.QueryResult`, generic structures, nested tree structures) and draw them. They don't have database-specific code.
2. **Implement Custom Catalog Queries**: Replace the PostgreSQL system catalog queries in [`internal/db/metadata/objects.go`](file:///home/mdfranz/github/pgtui/internal/db/metadata/objects.go) with equivalent queries for your target database:
   - **SQLite**: Query `sqlite_master` for tables/views/indices, and run `PRAGMA table_info(x)` for column schemas.
   - **ClickHouse**: Query `system.tables`, `system.columns`, `system.databases`, and `system.functions`.
   - **DuckDB**: Query `information_schema` tables.
   - **MongoDB**: Use `listCollections`, `$sample` for data preview, and the aggregation pipeline for schema inference.
3. **Customize Value Formatting**: Databases return different driver types. Implement a string formatting layer similar to `convertValueToString()` in [`internal/db/query/executor.go`](file:///home/mdfranz/github/pgtui/internal/db/query/executor.go) to handle target-specific collections, timestamps, geo types, or structs cleanly.
4. **Adapt Connection Discovery**: Swap env var names (`MYSQL_HOST`, `CLICKHOUSE_URL`), default ports, and config file paths (`~/.my.cnf`, `~/.clickhouse-client/config.xml`) in the discovery layer.
5. **Register Engine Commands**: Add engine-specific palette entries (e.g., `EXPLAIN ANALYZE`, `SHOW PROCESSLIST`, `DESCRIBE`) to the command registry at startup.
