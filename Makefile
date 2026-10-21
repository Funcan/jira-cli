BINARY = "jira"

.PHONY: default build test format lint ci clean

default: format build lint test

build:
	go build -o $(BINARY) main.go 

test:
	go test -cover ./...

format:
	go fmt ./...

lint:
	golangci-lint run

ci:
	@command -v act >/dev/null 2>&1 || { echo "act not found - install it from https://github.com/nektos/act (e.g. 'brew install act')"; exit 1; }
	act pull_request

clean:
	rm -f $(BINARY) coverage.out coverage.html
