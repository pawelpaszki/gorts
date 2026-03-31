.PHONY: test

# Run unit tests (untagged test files only, no cache)
test:
	go test -count=1 ./...
