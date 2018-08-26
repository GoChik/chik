package iosomething

import "fmt"

// Handler is the interface that handles network messages
// and optionally can return a reply
type Handler interface {
	fmt.Stringer
	HandlerRoutine(remote *Remote)
	Status() interface{}
}
