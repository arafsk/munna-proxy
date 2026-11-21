// Package version holds build metadata injected at link time.
package version

import "fmt"

var (
	// Version is the release version, for example "v0.1.0" or "dev".
	Version = "dev"
	// Commit is the short git commit the binary was built from.
	Commit = "none"
	// Date is the UTC build timestamp.
	Date = "unknown"
)

// String returns a human-readable version line.
func String() string {
	return fmt.Sprintf("munna-proxy %s (commit %s, built %s)", Version, Commit, Date)
}
