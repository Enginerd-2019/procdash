.PHONY: all fmt vet test cover clean

# The pre-commit sequence.
all: fmt vet test

# Rewrites files in place; prints the name of each file it changed.
fmt:
	gofmt -l -w .

vet:
	go vet ./...

# Fixture-driven; never needs root.
test:
	go test -v ./...

# Per-function coverage. Floors: pure parse functions 100%, package >= 80%.
cover:
	go test -coverprofile=cover.out ./...
	go tool cover -func=cover.out

clean:
	rm -f cover.out
