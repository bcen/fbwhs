// Package client defines client information when connected to the SSE broker.
package client

import (
	"fmt"
	"time"

	"github.com/davidsbond/sse/event"

	"github.com/rs/xid"
)

type (
	// The Client type represents a client connected to the broker.
	Client struct {
		id        string
		notify    chan *event.Event
		timeout   time.Duration
		failures  int
		tolerance int
	}
)

// New creates a new instance of the Client type using the provided timeout
// and tolerance. The 'timeout' parameter determines how long the client will attempt
// to write. The 'tolerance' parameter determines how many sequential errors the
// client will make before ShouldDisconnect returns true. The 'id' parameter allows
// you to specify a custom identifier for the client, if it is blank, a random
// identifier is created for the client.
func New(timeout time.Duration, tolerance int, id string) *Client {
	ret := &Client{
		id:        id,
		notify:    make(chan *event.Event),
		timeout:   timeout,
		failures:  0,
		tolerance: tolerance,
	}

	if id == "" {
		ret.id = xid.New().String()
	}

	return ret
}

// ID returns the client's unique identifier.
func (c *Client) ID() string {
	return c.id
}

// Listen reads event data from the broker.
func (c *Client) Listen() <-chan *event.Event {
	return c.notify
}

// Write attempts to write the provided event to the client. If writing
// exceeds the timeout, an error is returned.
func (c *Client) Write(evt *event.Event) error {
	select {
	case c.notify <- evt:
		c.failures = 0
		return nil
	case <-time.Tick(c.timeout):
		c.failures++
		return fmt.Errorf("failed to write to client %v, timeout exceeded", c.id)
	}
}

// ShouldDisconnect determines if a client has had too many sequential errors and
// should be forcefully disconnected from the broker.
func (c *Client) ShouldDisconnect() bool {
	return c.failures >= c.tolerance
}
