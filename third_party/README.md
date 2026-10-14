# third_party

faketandem needs no pumpX2 by default. A prebuilt pumpX2 `cliparser` jar placed
at `third_party/pumpx2-cliparser-<version>.jar` (and passed via
`-pumpx2-jar-path`) is for `-message-codec pumpx2`/`compare` and
`-jpake-mode pumpx2`, letting faketandem run them without a working Gradle/JDK
toolchain on the host (e.g. on a Raspberry Pi, where Gradle's own JDK-vs-JRE
toolchain detection has known bugs -- see gradle/gradle#30499). This directory
is not tracked in git (see `.gitignore`); place the jar here yourself.

## Obtaining the jar

Run `make jar` from the repo root -- it downloads a prebuilt
`pumpx2-cliparser-<version>.jar` from jitpack.io (see the `Makefile`'s
`PUMPX2_CLIPARSER_VERSION` for the pinned version) directly into this
directory. No local Gradle/JDK toolchain or pumpX2 checkout needed.

To bump the version, update `PUMPX2_CLIPARSER_VERSION` in the `Makefile` and
re-run `make jar`. A jitpack release can lag pumpX2's `dev` branch, which the
Go message types (`pkg/pumpx2/messages`) are checked against; build
the jar from that branch (`./gradlew :cliparser:shadowJar`) to compare like
with like.
