.PHONY: build test clean

K6_VERSION ?= v1.6.1

build:
	xk6 build $(K6_VERSION) --with github.com/henrikrexed/xk6-output-opentelemetry=.

test:
	go test -v -race ./...

clean:
	rm -f k6
