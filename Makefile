.PHONY: build jar all

PUMPX2_CLIPARSER_VERSION := v1.9.0
PUMPX2_CLIPARSER_JAR := third_party/pumpx2-cliparser-$(PUMPX2_CLIPARSER_VERSION:v%=%).jar
PUMPX2_CLIPARSER_URL := https://jitpack.io/com/github/jwoglom/pumpX2/pumpx2-cliparser/$(PUMPX2_CLIPARSER_VERSION)/pumpx2-cliparser-$(PUMPX2_CLIPARSER_VERSION).jar

build:
	GOOS=linux GOARCH=arm go build -o main ./

# Downloads a prebuilt cliparser jar from jitpack.io instead of building pumpX2
# from source -- no local Gradle/JDK toolchain needed (avoids the Gradle
# JDK-vs-JRE toolchain bugs seen on Raspbian, gradle/gradle#30499 et al).
jar:
	mkdir -p third_party
	curl -fL -o $(PUMPX2_CLIPARSER_JAR) $(PUMPX2_CLIPARSER_URL)

.DEFAULT_GOAL := all
all: jar build
