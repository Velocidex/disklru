all:
	go build -o ./inspector ./cmd

test:
	go test -v ./...
