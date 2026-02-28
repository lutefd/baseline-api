.PHONY: run test tidy migrate

run:
	go run ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy

migrate:
	go run ./cmd/migrate
