# third_party

`run.sh` expects a prebuilt pumpX2 `cliparser` jar at
`third_party/pumpx2-cliparser-<version>.jar` (passed via `-pumpx2-jar-path`),
letting faketandem run in JAR mode without needing a working Gradle/JDK
toolchain on the host (e.g. on a Raspberry Pi, where Gradle's own JDK-vs-JRE
toolchain detection has known bugs -- see gradle/gradle#30499). This directory
is not tracked in git (see `.gitignore`); place the jar here yourself.

## Obtaining the jar

Run `make jar` from the repo root -- it downloads a prebuilt
`pumpx2-cliparser-<version>.jar` from jitpack.io (see the `Makefile`'s
`PUMPX2_CLIPARSER_VERSION` for the pinned version) directly into this
directory. No local Gradle/JDK toolchain or pumpX2 checkout needed.

To bump the version, update `PUMPX2_CLIPARSER_VERSION` in the `Makefile` and
re-run `make jar`; also update `run.sh`'s `-pumpx2-jar-path` argument to match
the new filename.
