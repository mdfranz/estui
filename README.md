# estui

`estui` is an interactive Terminal User Interface (TUI) for querying Elasticsearch using **ES|QL** (Elasticsearch Query Language). It is designed to quickly inspect data, build filters dynamically, and browse query results from your terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

---

## Features

- **Time Picker**: Select a lookback window of 1, 5, 15, 30, or 60 minutes from a persistent bar at the top. The time filter is injected automatically before every query — no need to write `WHERE @timestamp` by hand.
- **Interactive Query Editor**: Multi-line editor with line numbers. Write pure ES|QL — SORT, LIMIT, and time filters are appended by the UI controls, not the editor.
- **Options Bar**: Controls for `LIMIT` (default 100) and `SORT` (field + ASC/DESC order) live below the editor and are appended to every query automatically.
- **Index Discovery**: Shows active indices from the selected time window on startup. Press `1`–`9` to load a pre-filled query template for the corresponding index.
- **Aggregation Suggestions**: Selecting an index expands up to four suggested aggregation queries tailored to the index schema. Press `a`–`d` to load one into the editor.
- **Dynamic Field Discovery**: Field types are discovered concurrently for all hinted indices so suggestions reflect the actual schema rather than hard-coded patterns.
- **Dynamic Table View**: Results hide columns containing only `null` or empty values automatically for improved readability.
- **Detailed Record View**: Examine individual rows field by field:
  - **Dynamic Filters**: Press `Enter` on any field to append `| WHERE field == value` to your query.
  - **Column Selection**: Toggle field selection with `Space` and press `Ctrl+K` to append `| KEEP field1, field2...` to your query.
  - **Quick Paging**: Browse adjacent rows using `n`/`p` or `[`/`]` without leaving the record detail view.
- **Query Cancellation**: Press `Esc` while a query is running to cancel execution.

---

## Keybindings

### Global
| Key | Action |
|-----|--------|
| `Tab` | Cycle focus: Time Picker → Editor → Options → Table → Time Picker |
| `Ctrl+R` / `Alt+Enter` | Run the current query |
| `Ctrl+C` | Quit |

### Time Picker (top bar)
| Key | Action |
|-----|--------|
| `←` / `→` (or `h` / `l`) | Move between time presets (1m, 5m, 15m, 30m, 60m) |
| `1`–`5` | Jump directly to the corresponding preset |
| `Enter` / `Tab` / `Esc` | Confirm selection and move to editor |

### Query Editor
| Key | Action |
|-----|--------|
| `1`–`9` | Load query template for the corresponding active index (only when editor is empty) |

### Options Bar (Limit / Sort)
| Key | Action |
|-----|--------|
| `Tab` | Cycle: Limit → Sort field → Sort order → Table |
| `Space` | Toggle sort order between `ASC` and `DESC` (when Sort Order is focused) |
| `Esc` | Return to editor |

### Results Table
| Key | Action |
|-----|--------|
| `Enter` / `v` | Open **Record View** for the selected row |
| `a`–`d` | Load the corresponding aggregation suggestion into the editor |
| `q` | Quit (only when table is focused) |

### Record View
| Key | Action |
|-----|--------|
| `j` / `k` (or `↑` / `↓`) | Scroll through fields |
| `Space` | Toggle field selection (marked with `✓`) |
| `Enter` | Append `\| WHERE field == value` to query and return to editor |
| `Ctrl+K` | Append `\| KEEP <selected fields>` to query and return to editor |
| `n` / `p` (or `[` / `]`) | Show next / previous row |
| `Esc` / `v` | Return to Results Table |

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
```bash
make build
```

### Run
```bash
export ELASTIC_URL="https://your-elasticsearch-host:9200"
export ELASTIC_KEY="your-api-key"
./estui
```
