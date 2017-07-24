// Package plugins defines a set of additionals routines that are agnostic from the
// application purpose, they are working like daemons in background during
// the application lifecycle
package plugins

// Plugin functions
type Plugin interface {
	Name() string
	Start()
	Stop()
}
