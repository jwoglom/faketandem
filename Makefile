.PHONY: build pumpx2

build:
	GOOS=linux GOARCH=arm go build -o main ./

pumpx2:
	@if [ ! -d pumpX2 ]; then \
		git clone https://github.com/jwoglom/pumpX2 pumpX2; \
	fi

.DEFAULT_GOAL := all
all: pumpx2 build
