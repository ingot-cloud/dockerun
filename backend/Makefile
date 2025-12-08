.PHONY: run build test clean

run:
	go run main.go

build:
	go build -o ../bin/dockerun main.go

test:
	go test ./...

clean:
	go clean
	rm -f ../bin/dockerun

