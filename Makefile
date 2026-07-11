.PHONY: build pumpx2 jar

build:
	GOOS=linux GOARCH=arm go build -o main ./

pumpx2:
	@if [ ! -d pumpX2 ]; then \
		git clone https://github.com/jwoglom/pumpX2 pumpX2; \
	fi

jar: pumpx2
	cd pumpX2 && ./gradlew :cliparser:shadowJar

.DEFAULT_GOAL := all
all: pumpx2 build
