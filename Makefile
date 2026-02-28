.PHONY: run test tidy

run:
	go run ./cmd/api

test:
	go test ./...

tidy:
	go mod tidy
