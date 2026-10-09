// Package appenv answers questions about the APP_ENV a process runs under.
//
// An environment name should be a label, not a switch that changes how a
// service loads configuration. The questions here are the few that have to
// depend on it, and each one is written to fail in its own safe direction
// when the name is one nobody recognises.
package appenv

// local are the names that mean "a developer's machine or a test run".
var local = map[string]bool{
	"":                true, // unset: a local run
	"local":           true,
	"test":            true,
	"integrationtest": true,
}

// docs are the names where API documentation is served.
var docs = map[string]bool{
	"":                true,
	"local":           true,
	"test":            true,
	"integrationtest": true,
	"development":     true,
	"qa":              true,
}

// IsDeployed reports whether name is a deployed environment, where required
// cross-service configuration must be present and its absence should stop the
// process.
//
// It is a deny-list of local names, not an allow-list of deployed ones. The
// safe mistake here is failing loudly, so an unrecognised name, a typo
// included, counts as deployed.
//
// Do not test for a substring instead. A name such as "development" is a
// deployed environment in practice, and a check for "dev" exempts it.
func IsDeployed(name string) bool { return !local[name] }

// DocsEnabled reports whether a process running under name should serve its
// API documentation.
//
// It is an allow-list, the opposite shape to IsDeployed, on purpose. The safe
// mistake here is serving nothing, so an unrecognised name gets no docs, and
// neither do "staging", "production" or "prod".
func DocsEnabled(name string) bool { return docs[name] }
