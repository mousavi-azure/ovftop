// Package version holds the application's semantic version
// (https://semver.org): MAJOR.MINOR.PATCH.
package version

// Version is a var, not a const, so release builds can pin it via
// -ldflags "-X github.com/mousavi-azure/ovftop/internal/version.Version=1.2.3" without touching
// source. Bump MAJOR for breaking changes (config/profile format,
// keybindings), MINOR for backwards-compatible features, PATCH for fixes.
var Version = "1.0.1"
