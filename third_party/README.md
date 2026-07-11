# third_party

`run.sh` expects a prebuilt pumpX2 `cliparser` jar at
`third_party/pumpx2-cliparser-<version>.jar` (passed via `-pumpx2-jar-path`),
letting faketandem run in JAR mode without needing a working Gradle/JDK
toolchain on the host (e.g. on a Raspberry Pi, where Gradle's own JDK-vs-JRE
toolchain detection has known bugs -- see gradle/gradle#30499). This directory
is not tracked in git (see `.gitignore`); place the jar here yourself.

## Obtaining the jar

The last one used here was `pumpx2-cliparser-1.9.0.jar`
(SHA-256 `04e2005e0707586330188ab8644ffe876ae00d6a11e803a59de04504c585159d`),
fetched from pumpX2's "Android CI" workflow's `maven-repository.zip` artifact
on the `dev` branch (commit `3250225`, run #398,
https://github.com/jwoglom/pumpX2/actions/runs/28298802885), extracted at
`com/jwoglom/pumpx2/pumpx2-cliparser/1.9.0/pumpx2-cliparser-1.9.0.jar`.

To get a current one:

1. Find the latest successful "Android CI" run on pumpX2's `dev` branch.
2. Download its `maven-repository.zip` artifact (or the `pumpx2-cliparser-all.jar`
   artifact directly, if that upload step succeeded for that run).
3. Extract `com/jwoglom/pumpx2/pumpx2-cliparser/<version>/pumpx2-cliparser-<version>.jar`
   and place it here.
4. Update `run.sh`'s `-pumpx2-jar-path` argument to match the filename.
