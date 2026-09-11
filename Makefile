.PHONY: run build tidy test snapshot

build:
	go build -o bin/api ./cmd

tidy:
	go mod tidy

test:
	go test ./...

run:
	go run ./cmd

snapshot:
	python3 python/runner.py data/01_full_constellation.json 0
