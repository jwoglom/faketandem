.PHONY: build
build:
	GOOS=linux GOARCH=arm go build -o main ./
