.PHONY: build pumpx2 jar

build:
	GOOS=linux GOARCH=arm go build -o main ./

pumpx2:
	@if [ ! -d pumpX2 ]; then \
		git clone https://github.com/jwoglom/pumpX2 pumpX2; \
	fi
	# Default org.gradle.jvmargs of -Xmx4096m is too large for a Raspberry Pi's RAM.
	sed -i 's/org.gradle.jvmargs=-Xmx4096m/org.gradle.jvmargs=-Xmx1024m/' pumpX2/gradle.properties
	# The configuration cache has a serialization bug with some vendor-patched JDK
	# builds (observed with Raspbian's OpenJDK 17) that crashes the whole build with
	# "does not provide the required capabilities: [JAVA_COMPILER]" while it's merely
	# trying to write the cache entry, not while actually compiling. Disable it; it's
	# only a speed optimization for repeated builds, not needed for a one-shot jar.
	sed -i 's/org.gradle.configuration-cache=true/org.gradle.configuration-cache=false/' pumpX2/gradle.properties
	# sampleapp/androidLib require the Android SDK and a working git subprocess to
	# even configure, which a headless Pi running just the cliparser jar has neither
	# of; drop them from the module graph so Gradle never evaluates them.
	sed -i "s/include ':sampleapp', ':androidLib', ':messages', ':shared', ':cliparser'/include ':messages', ':shared', ':cliparser'/" pumpX2/settings.gradle

jar: pumpx2
	cd pumpX2 && ./gradlew :cliparser:shadowJar

.DEFAULT_GOAL := all
all: pumpx2 build
