.PHONY: run build test clean

dev:
	TEMPLATE_DIR=../templates DATABASE_TYPE=mysql MYSQL_DSN="dockerun:dockerun123@tcp(localhost:3306)/dockerun?charset=utf8mb4&parseTime=True&loc=Local" go run main.go

build:
	go build -o ../bin/dockerun main.go

test:
	go test ./...

clean:
	go clean
	rm -f ../bin/dockerun

