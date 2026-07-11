.PHONY: build pumpx2 jar

build:
	GOOS=linux GOARCH=arm go build -o main ./

pumpx2:
	@if [ ! -d pumpX2 ]; then \
		git clone https://github.com/jwoglom/pumpX2 pumpX2; \
	fi
	# Default org.gradle.jvmargs of -Xmx4096m is too large for a Raspberry Pi's RAM.
	sed -i 's/org.gradle.jvmargs=-Xmx4096m/org.gradle.jvmargs=-Xmx1024m/' pumpX2/gradle.properties
	# The configuration cache is only a speed optimization for repeated builds, not
	# needed for a one-shot jar, and it obscures the real error underneath it when
	# something else goes wrong (as happened below), so disable it.
	sed -i 's/org.gradle.configuration-cache=true/org.gradle.configuration-cache=false/' pumpX2/gradle.properties
	# sampleapp/androidLib require the Android SDK and a working git subprocess to
	# even configure, which a headless Pi running just the cliparser jar has neither
	# of; drop them from the module graph so Gradle never evaluates them.
	sed -i "s/include ':sampleapp', ':androidLib', ':messages', ':shared', ':cliparser'/include ':messages', ':shared', ':cliparser'/" pumpX2/settings.gradle
	# Gradle 8.10+ has a known JDK-vs-JRE toolchain auto-detection bug
	# (gradle/gradle#30499, #30421, #30652) that can misjudge a real, working JDK as
	# lacking a compiler ("does not provide the required capabilities: [JAVA_COMPILER]"),
	# observed with Raspbian's OpenJDK 17 package. Pin org.gradle.java.home explicitly
	# to the JDK already resolved on PATH to bypass that detection entirely.
	@JAVA_HOME_RESOLVED="$$(dirname "$$(dirname "$$(readlink -f "$$(command -v javac)")")")"; \
	if grep -q '^org.gradle.java.home=' pumpX2/gradle.properties; then \
		sed -i "s|^org.gradle.java.home=.*|org.gradle.java.home=$$JAVA_HOME_RESOLVED|" pumpX2/gradle.properties; \
	else \
		echo "org.gradle.java.home=$$JAVA_HOME_RESOLVED" >> pumpX2/gradle.properties; \
	fi

jar: pumpx2
	cd pumpX2 && ./gradlew :cliparser:shadowJar

.DEFAULT_GOAL := all
all: pumpx2 build
