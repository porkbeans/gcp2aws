.PHONY: build test
build:
	go build -v

run: clean build
	./gcp2aws

test: clean
	go test -v -cover -coverprofile cover.out
	go tool cover -html cover.out -o cover.html

clean:
	rm -f gcp2aws
	rm -rf ~/.cache/gcp2aws
