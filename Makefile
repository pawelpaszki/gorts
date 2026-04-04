.PHONY: test test-integration test-e2e test-all

# Run unit tests (untagged test files only, no cache)
test:
	go test -count=1 ./...

# Run integration tests only (from test/integration folder)
test-integration:
	go test -v -count=1 -tags=integration ./test/integration/...

# Run e2e tests only (from test/e2e folder)
test-e2e:
	go test -v -count=1 -tags=e2e ./test/e2e/...

# Run all tests (unit + integration + e2e)
test-all: test test-integration test-e2e

