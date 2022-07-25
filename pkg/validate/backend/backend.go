// Package backend provides an interface for what methods a backend should include
package backend

import "github.com/chelnak/pdk/pkg/tool"

type Backend interface {
	Validate(tool *tool.Tool) error
	Exec(tool *tool.Tool) error
	Status() Status
}

// The Status must report whether the backend is available
// and any useful status information; in the case of the backend
// being unavailable, report the error message to the user.
type Status struct {
	IsAvailable bool
	StatusMsg   string
}
