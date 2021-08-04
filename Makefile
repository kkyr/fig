GOLANGCILINT=$(GOPATH)/bin/golangci-lint
$(GOLANGCILINT):
	curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.29.0

.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint: $(GOLANGCILINT)
	golangci-lint run ./...