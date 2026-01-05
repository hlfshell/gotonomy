# SemVer Package

A semantic versioning utility package for Go with comparison methods and build-time version support.

## Features

- Parse semantic version strings (supports `v1.2.3` or `1.2.3` format)
- Optional hash support (`v1.2.3-abc123` or `1.2.3:abc123`)
- Comparison methods: `Compare()`, `LT()`, `LTE()`, `GT()`, `GTE()`, `EQ()`
- Build-time version injection via `-ldflags`

## Basic Usage

```go
import "github.com/hlfshell/gotonomy/utils/semver"

// Parse a version string
v1, err := semver.NewSemVer("v1.2.3")
if err != nil {
    log.Fatal(err)
}

v2, _ := semver.NewSemVer("v1.2.4")

// Compare versions
if v1.LT(v2) {
    fmt.Println("v1 is older than v2")
}

// Or use Compare directly
switch v1.Compare(v2) {
case -1:
    fmt.Println("v1 < v2")
case 0:
    fmt.Println("v1 == v2")
case 1:
    fmt.Println("v1 > v2")
}
```

## Build-Time Version Injection

You can inject version information at build time using `-ldflags`:

```bash
go build -ldflags "\
  -X github.com/hlfshell/gotonomy/utils/semver.VersionInfo.Major=1 \
  -X github.com/hlfshell/gotonomy/utils/semver.VersionInfo.Minor=2 \
  -X github.com/hlfshell/gotonomy/utils/semver.VersionInfo.Patch=3 \
  -X github.com/hlfshell/gotonomy/utils/semver.VersionInfo.Hash=abc123"
```

Then in your code:

```go
version, err := semver.GetBuildVersion()
if err != nil {
    // Build-time version not set - handle error or use NewSemVer with hardcoded version
    version, _ = semver.NewSemVer("v1.0.0")
}
```

## Comparison Methods

All comparison methods ignore the hash and only compare major.minor.patch:

- `Compare(other SemVer) int` - Returns -1, 0, or 1
- `LT(other SemVer) bool` - Less than
- `LTE(other SemVer) bool` - Less than or equal
- `GT(other SemVer) bool` - Greater than
- `GTE(other SemVer) bool` - Greater than or equal
- `EQ(other SemVer) bool` - Equal

## Example: Using in Tool Cards

```go
// At build time, set version via ldflags
version, err := semver.GetBuildVersion()
if err != nil {
    // Fall back to hardcoded version if build-time version not set
    version, _ = semver.NewSemVer("v1.0.0")
}

// Use in tool definition
tool := NewTool(
    "my-tool",
    "Description",
    version, // SemVer instance
    // ... other parameters
)
```

