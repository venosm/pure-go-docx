.PHONY: tidy fmt test lint build vet

tidy:
	go mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

test:
	go test ./...

lint:
	go vet ./...

vet:
	go vet ./...

build:
	go build ./...
