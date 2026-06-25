package messages

type RunQueryMsg struct{}

type QueryResultMsg struct {
	Columns []string
	Rows    [][]string
	Err     error
}

type PingResultMsg struct {
	Err error
}

type IndexHint struct {
	Index string
	Count string
}

type HintsMsg struct {
	Hints []IndexHint
	Err   error
}
