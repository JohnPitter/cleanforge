.PHONY: test test-unit test-integration test-e2e test-all lint security

test-unit:
	go test ./internal/... -v -count=1

test-integration:
	go test ./internal/... -v -count=1 -tags=integration

test-e2e:
	go test ./tests/e2e/... -v -count=1 -tags=e2e -timeout=300s

test-all: test-unit test-integration test-e2e

test: test-unit

lint:
	golangci-lint run ./...

security:
	govulncheck ./...
	gosec ./...
