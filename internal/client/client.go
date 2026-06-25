package client

import (
	"errors"
	"os"

	"github.com/elastic/go-elasticsearch/v9"
)

func New() (*elasticsearch.TypedClient, error) {
	url := os.Getenv("ELASTIC_URL")
	key := os.Getenv("ELASTIC_KEY")
	if url == "" || key == "" {
		return nil, errors.New("ELASTIC_URL and ELASTIC_KEY must be set")
	}
	return elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{url},
		APIKey:    key,
	})
}
