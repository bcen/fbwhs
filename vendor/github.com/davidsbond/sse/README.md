# sse

[![CircleCI](https://circleci.com/gh/davidsbond/sse.svg?style=shield)](https://circleci.com/gh/davidsbond/sse)
[![Coverage Status](https://coveralls.io/repos/github/davidsbond/sse/badge.svg?branch=develop)](https://coveralls.io/github/davidsbond/sse?branch=develop)
[![GoDoc](https://godoc.org/github.com/davidsbond/sse?status.svg)](http://godoc.org/github.com/davidsbond/sse)
[![Go Report Card](https://goreportcard.com/badge/github.com/davidsbond/sse)](https://goreportcard.com/report/github.com/davidsbond/sse)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/davidsbond/sse/release/LICENSE)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fdavidsbond%2Fsse.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fdavidsbond%2Fsse?ref=badge_shield)

A golang library for implementing a [Server Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events) broker

## usage

```go
    // Create a configuration for the SSE broker
    config := sse.Config{
        Timeout: time.Duration * 3,
        Tolerance: 3,
        ErrorHandler: nil,
    }

    // Create a broker
    broker := sse.NewBroker(config)

    // Register the broker's HTTP handlers to your
    // router of choice. Route names are up to you.
    http.HandleFunc("/connect", broker.ClientHandler)
    http.HandleFunc("/broadcast", broker.EventHandler)

    // Programatically create events
    evt := sse.NewEvent("type", []byte("hello world"))

    broker.Broadcast(evt)
    broker.BroadcastTo("123", evt)

    // Programatically listen for events
    for evt := range broker.Listen() {
        // Do something
    }
```

## listening for events with [EventSource](https://developer.mozilla.org/en-US/docs/Web/API/EventSource)

```javascript
    // Connect to the event broker
    const source = new EventSource("http://localhost:8080/connect");

    // Optionally, supply a custom identifier for messaging individual clients
    // const source = new EventSource("http://localhost:8080/connect?id=1234");

    // Listen for incoming events
    source.onmessage = (event) => {
        // Do something with the event data
    };

    // Listen to specific event types
    source.addEventListener('event-type', (event) => {
        // Do something with the event data
    })
```

## creating events with HTTP

The event producing endpoint has two optional parameters, `type` & `id`. The `type` parameter specifies the type of event, these work similarly to topics in [Apache Kafka](https://kafka.apache.org/). The `id` parameter allows event producers to push an event to a specific client.

Here is an example for producing an event using the `curl` command.

```bash
  curl -H "Content-Type: application/json" -X POST -d '{"field":"value"}' http://localhost:8080/broadcast?type=event-type&id=123
```

## custom error handlers

If you want any HTTP errors returned to be in a certain format, you can supply a custom error handler to the broker

```go
    handler := func(w http.ResponseWriter, r *http.Request, err error) {
        // Write whatever you like to 'w'
    }

    // Create a configuration for the SSE broker
    config := sse.Config{
        Timeout: time.Duration * 3,
        Tolerance: 3,
        ErrorHandler: handler,
    }

    // Create a broker
    broker := sse.NewBroker(config)
```

## testing

The `sse.NewBroker` method returns an interface. Using this you can implement a mocked version of the broker to behave as you wish
in test scenarios. Below is an example using the [testify](https://github.com/stretchr/testify) package:

```go
package test

import (
    "net/http"
    "github.com/davidsbond/sse/event"
    "github.com/stretchr/testify/mock"
)

type (
    MockBroker {
        mock.Mock
    }
)

func (mb *MockBroker) Broadcast(evt *event.Event) error {
    args := mb.Called(evt)

    return args.Error(0)
}

func (mb *MockBroker) BroadcastTo(id string, evt *event.Event) error {
    args := mb.Called(id, evt)

    return args.Error(0)
}

func (mb *MockBroker) Listen() <-chan *event.Event {
    args := mb.Called()

    channel, ok := args.Get(0).(<-chan *event.Event)

    if !ok {
        return nil
    }

    return channel
}

func (mb *MockBroker) ClientHandler(w http.ResponseWriter, r *http.Request) {
    mb.Called(w, r)
}

func (mb *MockBroker) EventHandler(w http.ResponseWriter, r *http.Request) {
    mb.Called(w, r)
}
```

Using this mock, you can create expectations in your tests like this:

```go
    mock := new(MockBroker)

    // Determine the behaviour of the mock based on the id provided
    mock.On("BroadcastTo", "valid-id", mock.Anything).Returns(nil)
    mock.On("BroadcastTo", "invalid-id", mock.Anything).Returns(errors.New("error"))
```

With these expectations, you can assert whether or not your mocked methods have been called:

```go
    mock.AssertCalled(t, "BroadcastTo", "valid-id", mock.Anything)
    mock.AssertCalled(t, "BroadcastTo", "invalid-id", mock.Anything)
    mock.AssertNumberOfCalls(t, "BroadcastTo", 2)
```
