APP := letterpress

.PHONY: fmt test run tidy check

fmt:
	go fmt ./...

test:
	go test ./...

run:
	go run ./...

tidy:
	go mod tidy

check: fmt test
