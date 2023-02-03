.PHONY: test
test:
	go test -v ./...

.PHONY: lint
lint:
	./build/lint.sh ./...