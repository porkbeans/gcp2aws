.PHONY: update build run test coverage clean

gcp2aws: main.go
	make build

update:
	go get -u
	go mod tidy

build: clean
	go build -v

run: gcp2aws
	./gcp2aws

test: clean
	go test -v -cover -coverprofile cover.out

coverage: test
	go tool cover -html cover.out -o cover.html

clean:
	rm -f gcp2aws
	rm -rf ~/.cache/gcp2aws
	rm -f cover.out cover.html
