# ES|QL in the Elasticsearch Go Client

This guide summarizes the libraries and helper utilities available in the official Elasticsearch Go SDK for executing and mapping **ES|QL (Elasticsearch Query Language)** queries.

---

## 1. Key Libraries & Import Paths

To use ES|QL in your Go application, you primarily interact with two packages within the SDK:

### The Client Core
* **Package Path:** `github.com/elastic/go-elasticsearch/v9`
* **Purpose:** Provides the main `TypedClient` which contains the `Esql` namespace and request builder under `client.Esql.Query()`.
* **Go Symbol Reference:** [elasticsearch.go](file:///home/mdfranz/github/go-elasticsearch/elasticsearch.go)

### ES|QL Helpers
* **Package Path:** `github.com/elastic/go-elasticsearch/v9/typedapi/esql/query`
* **Purpose:** Houses mapping and iteration utilities to parse ES|QL responses into application-domain Go structs.
* **Go Symbol Reference:** [helpers.go](file:///home/mdfranz/github/go-elasticsearch/typedapi/esql/query/helpers.go)

---

## 2. Response Handling Styles

### A. Raw Query (Manual Parsing)
You can call the raw endpoint and receive response bytes formatted as JSON, CSV, TSV, or Text.
* **Method:** `client.Esql.Query().Query(queryString).Format("csv").Do(ctx)`

### B. Generic Object Mapping (`query.Helper`)
Automatically maps all row values returned by the ES|QL query into a slice of your custom struct `[]T`.
* **Signature:** `func Helper[T any](ctx context.Context, esqlQuery *Query) ([]T, error)`
* **Reference:** [Helper](file:///home/mdfranz/github/go-elasticsearch/typedapi/esql/query/helpers.go#L41-L88)

### C. Iterative Object Mapping (`query.NewIteratorHelper`)
Retrieves the response and allows lazy, row-by-row iteration using `More()` and `Next()`.
* **Signature:** `func NewIteratorHelper[T any](ctx context.Context, query *Query) (EsqlIterator[T], error)`
* **Reference:** [NewIteratorHelper](file:///home/mdfranz/github/go-elasticsearch/typedapi/esql/query/helpers.go#L142-L193)

---

## 3. Usage Example

Below is a complete implementation demonstrating both the standard collection helper and the streaming iterator helper:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/esql/query"
)

// Define your target domain struct mapping the columns in the ES|QL query
type Book struct {
	Name      string `json:"name"`
	Author    string `json:"author"`
	PageCount int    `json:"page_count"`
}

func main() {
	// Initialize the typed client
	client, err := elasticsearch.NewTyped()
	if err != nil {
		log.Fatalf("Error creating client: %s", err)
	}

	ctx := context.Background()
	queryString := `from library | where page_count > 200 | sort name asc | limit 10`

	// ==========================================
	// Option 1: Map all results at once (Helper)
	// ==========================================
	qry1 := client.Esql.Query().Query(queryString)
	books, err := query.Helper[Book](ctx, qry1)
	if err != nil {
		log.Fatalf("Error using Helper: %s", err)
	}

	fmt.Println("--- Helper Results ---")
	for _, book := range books {
		fmt.Printf(" * %s by %s (%d pages)\n", book.Name, book.Author, book.PageCount)
	}

	// ==========================================
	// Option 2: Stream results (IteratorHelper)
	// ==========================================
	qry2 := client.Esql.Query().Query(queryString)
	iterator, err := query.NewIteratorHelper[Book](ctx, qry2)
	if err != nil {
		log.Fatalf("Error creating iterator: %s", err)
	}

	fmt.Println("\n--- Iterator Results ---")
	for iterator.More() {
		book, err := iterator.Next()
		if err != nil {
			log.Fatalf("Iterator error: %s", err)
		}
		fmt.Printf(" * %s by %s (%d pages)\n", book.Name, book.Author, book.PageCount)
	}
}
```
