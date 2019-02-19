// Package broker contains types to be used to host a Server Sent Events broker.
package broker

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/davidsbond/sse/client"
	"github.com/davidsbond/sse/event"
)

type (
	// The Broker interface describes the Server Side Events broker, propagating messages
	// to all connected clients.
	Broker interface {
		Broadcast(evt *event.Event) error
		BroadcastTo(id string, evt *event.Event) error
		Listen() <-chan *event.Event
		ClientHandler(w http.ResponseWriter, r *http.Request)
		EventHandler(w http.ResponseWriter, r *http.Request)
	}

	// ErrorHandler is a convenience wrapper for the HTTP error handling function.
	ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

	defaultBroker struct {
		timeout      time.Duration
		clients      *sync.Map
		errorHandler ErrorHandler
		tolerance    int
	}
)

// New creates a new instance of the Broker type. The 'timeout' parameter determines how long
// the broker will wait to write a message to a client, if this timeout is exceeded, the client
// will not receive that message. The 'tolerance' parameter indicates how many sequential errors
// can occur when communicating with a client until the client is forcefully disconnected. The
// 'eh' parameter is a custom HTTP error handler that the broker will use when HTTP errors are
// raised. If 'eh' is null, the default http.Error method is used.
func New(timeout time.Duration, tolerance int, eh ErrorHandler) Broker {
	return &defaultBroker{
		timeout:      timeout,
		clients:      &sync.Map{},
		tolerance:    tolerance,
		errorHandler: eh,
	}
}

// Listen returns messages sent to the broker.
func (b *defaultBroker) Listen() <-chan *event.Event {
	client := client.New(b.timeout, b.tolerance, "")

	b.addClient(client)
	return client.Listen()
}

// BroadcastTo attempts to write a given payload to a specific client. If that client
// is not connected, an error is returned. The 'id' parameter determines the unique identifier
// of the client you're attempting to write to. The 'data' parameter determines the payload to
// send to that client.
func (b *defaultBroker) BroadcastTo(id string, evt *event.Event) error {
	client, err := b.getClient(id)

	if err != nil {
		return err
	}

	return client.Write(evt)
}

// Broadcast writes the given event to all connected clients. If a client exceeds its error tolerance, it is
// forcefully disconnected from the broker. All errors are concatenated with newlines and returned from this
// method as a single error.
func (b *defaultBroker) Broadcast(evt *event.Event) error {
	var out []string

	// Loop through each connected client.
	b.clients.Range(func(key, value interface{}) bool {
		client, ok := value.(*client.Client)

		// If we couldn't cast the client, something strange has
		// gotten into the map. Add an error to the array and
		// force disconnect the client.
		if !ok {
			err := fmt.Errorf("found malformed client with id %v, disconnecting", key)
			out = append(out, err.Error())
			b.removeClient(key)

			return true
		}

		// Attempt to write the event to the client
		if err := client.Write(evt); err != nil {
			// If an error occurred, check if we should force
			// disconnect the client.
			if client.ShouldDisconnect() {
				b.removeClient(client.ID())
			}

			out = append(out, err.Error())
		}

		return true
	})

	// If we have multiple errors, concatenate them with newlines.
	if len(out) > 0 {
		return errors.New(strings.Join(out, "\n"))
	}

	return nil
}

// EventHandler is an HTTP handler that allows a client to broadcast an event to the
// broker. This method should be registered to an endpoint of your choosing. For information
// on error handling, see the broker.SetErrorHandler method.
//
// Example using http (https://golang.org/pkg/net/http/)
//
// http.HandleFunc("/broadcast", broker.EventHandler)
// http.ListenAndServe(":8080")
//
// Example using Mux (https://github.com/gorilla/mux)
//
// r := mux.NewRouter()
// r.HandleFunc("/broadcast", broker.EventHandler).Methods("POST")
//
// http.ListenAndServe(":8080", r)
func (b *defaultBroker) EventHandler(w http.ResponseWriter, r *http.Request) {
	// Attempt to read the provided event data.
	data, err := ioutil.ReadAll(r.Body)

	// If we fail to read, either use the custom error handler or
	// use the default http error.
	if err != nil {
		b.httpError(w, r, err, http.StatusInternalServerError)
		return
	}

	if len(data) == 0 {
		err := errors.New("no event data provided")
		b.httpError(w, r, err, http.StatusBadRequest)
		return
	}

	// Obtain the client id if supplied
	id := r.URL.Query().Get("id")
	typ := r.URL.Query().Get("type")

	evt := event.New(typ, data)

	if id != "" {
		// Attempt to broadcast the event data to the specified client.
		err = b.BroadcastTo(id, evt)
	} else {
		// Attempt to broadcast the event data to all connected clients.
		err = b.Broadcast(evt)
	}

	if err != nil {
		b.httpError(w, r, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ClientHandler is an HTTP handler that allows a client to connect to the
// broker. This method should be registered to an endpoint of your choosing.
// For information on error handling, see the broker.SetErrorHandler method.
//
// Example using http (https://golang.org/pkg/net/http/)
//
// http.HandleFunc("/connect", broker.ClientHandler)
// http.ListenAndServe(":8080")
//
// Example using Mux (https://github.com/gorilla/mux)
//
// r := mux.NewRouter()
// r.HandleFunc("/connect", broker.ClientHandler).Methods("GET")
//
// http.ListenAndServe(":8080", r)
func (b *defaultBroker) ClientHandler(w http.ResponseWriter, r *http.Request) {
	// Attempt to cast the response writer to a flusher & close notifier
	flusher, fOK := w.(http.Flusher)
	notify, nOK := w.(http.CloseNotifier)

	if !(nOK && fOK) {
		// If we fail to cast, return an error.
		err := errors.New("client does not support streaming")

		b.httpError(w, r, err, http.StatusInternalServerError)
		return
	}

	// Set the required headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a new client with the configured timeout &
	// tolerance.
	client := client.New(b.timeout, b.tolerance, r.URL.Query().Get("id"))
	id := client.ID()

	// Ensure that no custom identifiers collide.
	if b.hasClient(id) {
		err := fmt.Errorf("a client with id %v already exists", id)

		b.httpError(w, r, err, http.StatusInternalServerError)
		return
	}

	b.addClient(client)

	// While the client is connected
	for {
		select {
		// If we read an event, write it to the client
		case event := <-client.Listen():
			fmt.Fprint(w, event.String())
			flusher.Flush()
			break

		// If we exceed the timeout, continue.
		case <-time.Tick(b.timeout):
			continue

		// If the connection is closed, disconnect the client.
		case <-notify.CloseNotify():
			b.removeClient(id)
			return
		}
	}
}

func (b *defaultBroker) addClient(client *client.Client) {
	b.clients.Store(client.ID(), client)
}

func (b *defaultBroker) removeClient(id interface{}) {
	b.clients.Delete(id)
}

func (b *defaultBroker) hasClient(id interface{}) bool {
	_, ok := b.clients.Load(id)

	return ok
}

func (b *defaultBroker) httpError(w http.ResponseWriter, r *http.Request, err error, code int) {
	if b.errorHandler != nil {
		b.errorHandler(w, r, err)
		return
	}

	http.Error(w, err.Error(), code)
}

func (b *defaultBroker) getClient(id interface{}) (*client.Client, error) {
	item, ok := b.clients.Load(id)

	if !ok {
		return nil, fmt.Errorf("no client with id %v exists", id)
	}

	client, ok := item.(*client.Client)

	if !ok {
		b.removeClient(id)
		return nil, fmt.Errorf("client with id %v is malformed, disconnecting", id)
	}

	return client, nil
}
