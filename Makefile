.PHONY: run build tidy test snapshot docker-build docker-run

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

docker-build:
	docker compose build

docker-run:
	docker compose up --build
