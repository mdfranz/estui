package app

import (
	"fmt"
	"strings"

	"github.com/mdfranz/estui/internal/messages"
)

type suggestion struct {
	label string
	query string
}

// suggestionsFor returns suggestions for an index. If field discovery has
// completed (cols non-nil), dynamic type-driven queries are generated.
// Known ECS data stream patterns are always tried first for richer queries.
func suggestionsFor(index string, cols []messages.ColumnInfo) []suggestion {
	if s := patternSuggestions(index); s != nil {
		return s
	}
	if len(cols) > 0 {
		return dynamicSuggestions(index, cols)
	}
	return genericSuggestions(index)
}

// patternSuggestions returns curated queries for known ECS data stream types.
// SORT and LIMIT are intentionally omitted — the options bar appends them.
// Returns nil if the index doesn't match any known pattern.
func patternSuggestions(index string) []suggestion {
	name := strings.ToLower(index)
	base := fmt.Sprintf("FROM %s", index)

	switch {
	case strings.Contains(name, "network_traffic.flow"):
		return []suggestion{
			{"Top source IPs", base + "\n| STATS count = COUNT(*) BY source.ip"},
			{"Top dest IPs + ports", base + "\n| STATS count = COUNT(*) BY destination.ip, destination.port"},
			{"Traffic by protocol", base + "\n| STATS count = COUNT(*) BY network.protocol"},
			{"Bytes by source IP", base + "\n| STATS total_bytes = SUM(network.bytes) BY source.ip"},
		}
	case strings.Contains(name, "network_traffic.dns"):
		return []suggestion{
			{"Top queried domains", base + "\n| STATS count = COUNT(*) BY dns.question.name"},
			{"Top DNS sources", base + "\n| STATS count = COUNT(*) BY source.ip"},
			{"Queries by record type", base + "\n| STATS count = COUNT(*) BY dns.question.type"},
			{"NXDOMAIN sources", base + "\n| WHERE dns.response_code == \"NXDOMAIN\"\n| STATS count = COUNT(*) BY source.ip"},
		}
	case strings.Contains(name, "endpoint.events.process"):
		return []suggestion{
			{"Top processes", base + "\n| STATS count = COUNT(*) BY process.name"},
			{"Processes by user", base + "\n| STATS count = COUNT(*) BY user.name, process.name"},
			{"Parent → child", base + "\n| STATS count = COUNT(*) BY process.parent.name, process.name"},
			{"Processes by host", base + "\n| STATS count = COUNT(*) BY host.name, process.name"},
		}
	case strings.Contains(name, "endpoint.events.file"):
		return []suggestion{
			{"Top file paths", base + "\n| STATS count = COUNT(*) BY file.path"},
			{"File ops by action", base + "\n| STATS count = COUNT(*) BY event.action"},
			{"Writes by process", base + "\n| WHERE event.action == \"creation\" OR event.action == \"modification\"\n| STATS count = COUNT(*) BY process.name"},
			{"File activity by host", base + "\n| STATS count = COUNT(*) BY host.name"},
		}
	case strings.Contains(name, "endpoint.events.network"):
		return []suggestion{
			{"Top dest IPs", base + "\n| STATS count = COUNT(*) BY destination.ip"},
			{"Top dest ports", base + "\n| STATS count = COUNT(*) BY destination.port"},
			{"Connections by process", base + "\n| STATS count = COUNT(*) BY process.name, destination.ip"},
			{"External connections", base + "\n| WHERE NOT STARTS_WITH(destination.ip, \"10.\") AND NOT STARTS_WITH(destination.ip, \"192.168.\")\n| STATS count = COUNT(*) BY destination.ip"},
		}
	case strings.Contains(name, "zeek.connection"):
		return []suggestion{
			{"Top talkers", base + "\n| STATS count = COUNT(*) BY source.ip"},
			{"Top dest IPs + ports", base + "\n| STATS count = COUNT(*) BY destination.ip, destination.port"},
			{"By service", base + "\n| STATS count = COUNT(*) BY zeek.connection.service"},
			{"Bytes by pair", base + "\n| STATS total = SUM(network.bytes) BY source.ip, destination.ip"},
		}
	case strings.Contains(name, "zeek.dns"):
		return []suggestion{
			{"Top queried domains", base + "\n| STATS count = COUNT(*) BY dns.question.name"},
			{"By query type", base + "\n| STATS count = COUNT(*) BY dns.question.type"},
			{"Top resolvers", base + "\n| STATS count = COUNT(*) BY source.ip"},
		}
	case strings.Contains(name, "metrics-system.diskio"):
		return []suggestion{
			{"Top disks by reads", base + "\n| STATS reads = SUM(system.diskio.read.bytes) BY system.diskio.name, host.name"},
			{"Top disks by writes", base + "\n| STATS writes = SUM(system.diskio.write.bytes) BY system.diskio.name, host.name"},
			{"I/O ops by host", base + "\n| STATS ops = SUM(system.diskio.read.ops) + SUM(system.diskio.write.ops) BY host.name"},
		}
	case strings.Contains(name, "metrics-system.network"):
		return []suggestion{
			{"Bytes in by interface", base + "\n| STATS bytes_in = SUM(system.network.in.bytes) BY system.network.name, host.name"},
			{"Bytes out by interface", base + "\n| STATS bytes_out = SUM(system.network.out.bytes) BY system.network.name, host.name"},
			{"Errors by interface", base + "\n| STATS errors = SUM(system.network.in.errors) + SUM(system.network.out.errors) BY system.network.name"},
		}
	case strings.Contains(name, "metrics-system.process"):
		return []suggestion{
			{"Top CPU consumers", base + "\n| STATS cpu = AVG(system.process.cpu.pct) BY system.process.name"},
			{"Top memory consumers", base + "\n| STATS mem = AVG(system.process.memory.pct) BY system.process.name"},
			{"Process count by host", base + "\n| STATS count = COUNT_DISTINCT(system.process.name) BY host.name"},
		}
	case strings.Contains(name, "metrics-system.cpu"):
		return []suggestion{
			{"CPU over time", base + "\n| STATS avg_cpu = AVG(system.cpu.total.pct) BY bucket = DATE_TRUNC(1 minute, @timestamp), host.name"},
			{"CPU by host", base + "\n| STATS avg_cpu = AVG(system.cpu.total.pct) BY host.name"},
		}
	}
	return nil
}

// dynamicSuggestions generates suggestions by inspecting actual field types.
// Works for any index schema. SORT and LIMIT omitted — appended by options bar.
func dynamicSuggestions(index string, cols []messages.ColumnInfo) []suggestion {
	base := fmt.Sprintf("FROM %s", index)

	var ipFields, numericFields, keywordFields []string
	for _, c := range cols {
		switch c.Type {
		case "ip":
			ipFields = append(ipFields, c.Name)
		case "keyword", "text":
			keywordFields = append(keywordFields, c.Name)
		case "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float":
			numericFields = append(numericFields, c.Name)
		}
	}

	interesting := rankKeywords(keywordFields)
	var sugs []suggestion

	for _, f := range ipFields {
		if len(sugs) >= 2 {
			break
		}
		sugs = append(sugs, suggestion{
			fmt.Sprintf("Top %s", f),
			fmt.Sprintf("%s\n| STATS count = COUNT(*) BY %s", base, f),
		})
	}

	for _, f := range interesting {
		if len(sugs) >= 3 {
			break
		}
		sugs = append(sugs, suggestion{
			fmt.Sprintf("By %s", f),
			fmt.Sprintf("%s\n| STATS count = COUNT(*) BY %s", base, f),
		})
	}

	if len(numericFields) > 0 && len(sugs) < 4 {
		f := numericFields[0]
		sugs = append(sugs, suggestion{
			fmt.Sprintf("Avg/max %s", f),
			fmt.Sprintf("%s\n| STATS avg_val = AVG(%s), max_val = MAX(%s) BY host.name", base, f, f),
		})
	}

	if len(sugs) < 4 {
		sugs = append(sugs, suggestion{
			"Events over time",
			base + "\n| STATS count = COUNT(*) BY bucket = DATE_TRUNC(1 minute, @timestamp)",
		})
	}

	if len(sugs) > 4 {
		sugs = sugs[:4]
	}
	return sugs
}

// genericSuggestions is the fallback when neither pattern nor field data is available.
func genericSuggestions(index string) []suggestion {
	base := fmt.Sprintf("FROM %s", index)
	return []suggestion{
		{"Events over time", base + "\n| STATS count = COUNT(*) BY bucket = DATE_TRUNC(1 minute, @timestamp)"},
		{"Events by host", base + "\n| STATS count = COUNT(*) BY host.name"},
		{"Events by category", base + "\n| STATS count = COUNT(*) BY event.category, event.type"},
		{"Events by agent", base + "\n| STATS count = COUNT(*) BY agent.type"},
	}
}

// rankKeywords scores keyword fields by their last path segment and returns
// them ordered by relevance for use as STATS BY keys.
func rankKeywords(fields []string) []string {
	highValue := map[string]int{
		"name": 10, "type": 10, "category": 10, "action": 9, "protocol": 9,
		"status": 8, "level": 8, "kind": 8, "method": 8, "code": 7,
		"reason": 7, "provider": 7, "outcome": 7, "severity": 6,
	}

	type scored struct {
		field string
		score int
	}
	var ranked []scored
	for _, f := range fields {
		parts := strings.Split(f, ".")
		score := highValue[parts[len(parts)-1]]
		if len(parts) > 4 {
			score -= 3
		}
		if score > 0 {
			ranked = append(ranked, scored{f, score})
		}
	}

	for i := 1; i < len(ranked); i++ {
		for j := i; j > 0 && ranked[j].score > ranked[j-1].score; j-- {
			ranked[j], ranked[j-1] = ranked[j-1], ranked[j]
		}
	}

	result := make([]string, len(ranked))
	for i, s := range ranked {
		result[i] = s.field
	}
	return result
}
