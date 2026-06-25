BIN := estui

.PHONY: build clean

build:
	go build -o $(BIN) ./cmd/estui

clean:
	rm -f $(BIN)
