.PHONY: tidy fmt test bench lint build vet

BENCH ?= .
BENCH_COUNT ?= 1

tidy:
	go mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

test:
	go test ./...

bench:
	go test -run '^$$' -bench '$(BENCH)' -benchmem -count $(BENCH_COUNT) ./...

lint:
	go vet ./...

vet:
	go vet ./...

build:
	go build ./...
