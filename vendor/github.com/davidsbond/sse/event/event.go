package event

import (
	"fmt"
)

type (
	// The Event type represents a Server Sent Event message.
	Event struct {
		data []byte
		typ  string
	}
)

// New creates a new event with the given type and data.
func New(t string, data []byte) *Event {
	return &Event{
		typ:  t,
		data: data,
	}
}

func (e *Event) String() string {
	return fmt.Sprintf("event:%s\ndata:%s\n\n", e.typ, e.data)
}
