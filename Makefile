.PHONY: test test-integration test-all

# Run unit tests (untagged test files only, no cache)
test:
	go test -count=1 ./...

# Run integration tests only (from test/integration folder)
test-integration:
	go test -v -count=1 -tags=integration ./test/integration/...

