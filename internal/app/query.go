package app

import (
	"context"
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/esqlformat"
	"github.com/mdfranz/estui/internal/messages"
)

type esqlResponse struct {
	Columns []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"columns"`
	Values [][]any `json:"values"`
}

const hintsQuery = `FROM * METADATA _index | WHERE @timestamp >= NOW() - 15 minutes | STATS count = COUNT(*) BY _index | SORT count DESC | LIMIT 10`

func loadHints(ctx context.Context, client *elasticsearch.TypedClient) tea.Cmd {
	return func() tea.Msg {
		raw, err := client.Esql.Query().
			Query(hintsQuery).
			Format(esqlformat.Json).
			Do(ctx)
		if err != nil {
			return messages.HintsMsg{Err: err}
		}

		var resp esqlResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return messages.HintsMsg{Err: err}
		}

		var hints []messages.IndexHint
		for _, row := range resp.Values {
			if len(row) < 2 {
				continue
			}
			count := fmt.Sprintf("%v", row[0])
			index := fmt.Sprintf("%v", row[1])
			hints = append(hints, messages.IndexHint{Index: index, Count: count})
		}
		return messages.HintsMsg{Hints: hints}
	}
}

func discoverFields(ctx context.Context, client *elasticsearch.TypedClient, index string) tea.Cmd {
	return func() tea.Msg {
		q := fmt.Sprintf("FROM %s | WHERE @timestamp >= NOW() - 15 minutes | LIMIT 1", index)
		raw, err := client.Esql.Query().
			Query(q).
			Format(esqlformat.Json).
			Do(ctx)
		if err != nil {
			return messages.FieldDiscoveryMsg{Index: index, Err: err}
		}
		var resp esqlResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return messages.FieldDiscoveryMsg{Index: index, Err: err}
		}
		cols := make([]messages.ColumnInfo, len(resp.Columns))
		for i, c := range resp.Columns {
			cols[i] = messages.ColumnInfo{Name: c.Name, Type: c.Type}
		}
		return messages.FieldDiscoveryMsg{Index: index, Columns: cols}
	}
}

func ping(ctx context.Context, client *elasticsearch.TypedClient) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Info().Do(ctx)
		return messages.PingResultMsg{Err: err}
	}
}

func runQuery(ctx context.Context, client *elasticsearch.TypedClient, esql string) tea.Cmd {
	return func() tea.Msg {
		raw, err := client.Esql.Query().
			Query(esql).
			Format(esqlformat.Json).
			Do(ctx)
		if err != nil {
			return messages.QueryResultMsg{Err: err}
		}

		var resp esqlResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return messages.QueryResultMsg{Err: fmt.Errorf("parse response: %w", err)}
		}

		cols := make([]string, len(resp.Columns))
		for i, c := range resp.Columns {
			cols[i] = c.Name
		}

		rows := make([][]string, len(resp.Values))
		for i, row := range resp.Values {
			cells := make([]string, len(row))
			for j, v := range row {
				if v == nil {
					cells[j] = "null"
				} else {
					cells[j] = fmt.Sprintf("%v", v)
				}
			}
			rows[i] = cells
		}

		return messages.QueryResultMsg{Columns: cols, Rows: rows}
	}
}
