.PHONY: build pumpx2 jar

build:
	GOOS=linux GOARCH=arm go build -o main ./

pumpx2:
	@if [ ! -d pumpX2 ]; then \
		git clone https://github.com/jwoglom/pumpX2 pumpX2; \
	fi
	# Default org.gradle.jvmargs of -Xmx4096m is too large for a Raspberry Pi's RAM.
	sed -i 's/org.gradle.jvmargs=-Xmx4096m/org.gradle.jvmargs=-Xmx1024m/' pumpX2/gradle.properties

jar: pumpx2
	cd pumpX2 && ./gradlew :cliparser:shadowJar

.DEFAULT_GOAL := all
all: pumpx2 build
