.PHONY: run build test clean

dev:
	go run main.go

build:
	go build -o ../bin/dockerun main.go

test:
	go test ./...

clean:
	go clean
	rm -f ../bin/dockerun

