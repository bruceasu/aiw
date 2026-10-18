# Third-party sources

Only the non-test source packages needed by this program are included. The
package source files are unchanged. The local go.mod files retain the module
names and remove requirements used only by excluded tools and other formats.
No dependency is fetched when this program is compiled or run.

| Module | Version | Source checksum | License |
| --- | --- | --- | --- |
| github.com/dsnet/compress | v0.0.1 | h1:PlZu0n3Tuv04TzpfPbrnI0HW/YwodEXDS+oPKahKF0Q= | BSD-3-Clause, third_party/dsnet-compress/LICENSE.md |
| golang.org/x/text | v0.24.0 | h1:dd5Bzh4yt5KYA8f9CJHCP4FB4D51c2c6JvN37xJJkJ0= | BSD-3-Clause, third_party/x-text/LICENSE |

The BZIP2 source was obtained by the explicitly authorized fixed-version
download. The encoding source came from the existing local Go module cache.
