# estui

`estui` is an interactive Terminal User Interface (TUI) for querying Elasticsearch using **ES|QL** (Elasticsearch Query Language). It is designed to quickly inspect data, build filters dynamically, and browse query results from your terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

---

## Features

- **Interactive Query Editor**: Multi-line editor with line numbers and running state indicators.
- **Index Discovery**: Shows active indices from the last 15 minutes on startup, letting you load pre-filled query templates using keyboard shortcuts `1`–`9`.
- **Dynamic Table View**: Clean tabular presentation of results that automatically hides columns containing only `null` or empty values for improved readability.
- **Detailed Record View**: Examine individual rows field by field:
  - **Dynamic Filters**: Press `Enter` on any field to append `| WHERE field == value` to your query.
  - **Column Selection**: Toggle field selection with `Space` and press `Ctrl+K` to append `| KEEP field1, field2...` to your query.
  - **Quick Paging**: Browse adjacent rows using `n`/`p` or `[`/`]` keys without leaving the record detail view.
- **Query Cancellation**: Press `Esc` while a query is running to cancel execution.

---

## Keybindings

### Global & Focus
* `Tab` - Toggle focus between the Query Editor and the Results Table
* `Ctrl+C` - Quit the application

### Query Editor
* `Ctrl+R` or `Alt+Enter` - Run the current query
* `1`–`9` - Load query template for the corresponding active index (only when the query input is empty)

### Results Table
* `Enter` or `v` - Open the detailed **Record View** for the selected row
* `q` - Quit the application (only when table is focused)

### Record View
* `j` / `k` (or `Up` / `Down` arrows) - Scroll through fields
* `Space` - Toggle field selection (marked with a `✓`)
* `Enter` - Append `| WHERE field == value` to the query and return to editor
* `Ctrl+K` - Append `| KEEP <selected fields>` to the query and return to editor
* `n` / `p` (or `[` / `]`) - Show next/previous row details
* `Esc` or `v` - Return to the **Results Table**

---

## Configuration

`estui` connects to Elasticsearch using two environment variables:

| Variable | Description |
|----------|-------------|
| `ELASTIC_URL` | The URL of your Elasticsearch cluster (e.g., `https://localhost:9200`) |
| `ELASTIC_KEY` | The API Key to authenticate with the cluster |

---

## Installation & Running

### Prerequisites
* Go 1.26 or higher

### Build
To compile the `estui` binary:
```bash
make build
```

To clean build artifacts:
```bash
make clean
```

### Run
Set the environment variables and run the compiled binary:
```bash
export ELASTIC_URL="https://your-elasticsearch-host:9200"
export ELASTIC_KEY="your-api-key"
./estui
```
