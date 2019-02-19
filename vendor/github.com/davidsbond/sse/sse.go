package sse

import (
	"time"

	"github.com/davidsbond/sse/event"

	"github.com/davidsbond/sse/broker"
)

type (
	// The Config type contains configuration variables for the SSE broker.
	Config struct {
		Timeout      time.Duration       // Determines how long the broker will wait to write to a client.
		Tolerance    int                 // Determines how many sequential errors a client can have until they are forcefully disconnected.
		ErrorHandler broker.ErrorHandler // Defines a custom HTTP error handling method to use when controller errors occur.
	}
)

// NewBroker creates a new instance of the SSE broker using the given configuration.
func NewBroker(cnf Config) broker.Broker {
	broker := broker.New(cnf.Timeout, cnf.Tolerance, cnf.ErrorHandler)

	return broker
}

// NewEvent creates a new event that can be broadcast to clients.
func NewEvent(t string, data []byte) *event.Event {
	return event.New(t, data)
}
