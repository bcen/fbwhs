package internal_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"fbwhs/internal"
	"github.com/davidsbond/sse"
	"github.com/davidsbond/sse/event"
)

type inMemBroker struct {
	events []*event.Event
}

func (b *inMemBroker) Broadcast(evt *event.Event) error {
	// Not implemented
	return nil
}

func (b *inMemBroker) BroadcastTo(id string, evt *event.Event) error {
	b.events = append(b.events, evt)
	return nil
}

func (b *inMemBroker) Listen() <-chan *event.Event {
	// Not implemented
	return nil
}

func (b *inMemBroker) ClientHandler(w http.ResponseWriter, r *http.Request) {
	// Not implemented
}

func (b *inMemBroker) EventHandler(w http.ResponseWriter, r *http.Request) {
	// Not implemented
}

func TestSubscription(t *testing.T) {
	wh := internal.NewDefaultWebhookHandler()
	wid := "abc123"
	id1, _ := wh.Subscribe(wid)
	id2, _ := wh.Subscribe(wid)
	if id1 == id2 {
		t.Errorf("Each subscription should be unique")
	}
}

func TestUnsubscribe(t *testing.T) {
	wh := internal.NewDefaultWebhookHandler()
	wid := "abc123"
	eventID, _ := wh.Subscribe(wid)
	if !wh.Unsubscribe(eventID) {
		t.Errorf("Should return true if subscribed")
	}
}

func TestUnsubscribeDeleted(t *testing.T) {
	wh := internal.NewDefaultWebhookHandler()
	if wh.Unsubscribe("deleted eventID") {
		t.Errorf("Should return false for deleted")
	}

	eventID := "abc123"
	wh.Subscribe(eventID)
	wh.Unsubscribe(eventID)
	if wh.Unsubscribe(eventID) {
		t.Errorf("Should return false for deleted")
	}
}

func TestGetEventIDs(t *testing.T) {
	wh := internal.NewDefaultWebhookHandler()
	wid := "abc123"
	wh.Subscribe(wid)
	if len(wh.EventIDs(wid)) == 0 {
		t.Errorf("Should have subscriptions")
	}

	if len(wh.EventIDs("foo")) != 0 {
		t.Errorf("Should not have subscriptions")
	}
}

func TestForward(t *testing.T) {
	b := &inMemBroker{}
	wh := internal.NewWebhookHandler(b)
	wid := "abc123"
	wh.Subscribe(wid)
	if wh.Forward(wid, nil, "a body") != nil {
		t.Fail()
	}

	w := internal.Webhook{Header: nil, Body: "a body"}
	jsonBody, _ := json.Marshal(w)
	event := sse.NewEvent("webhook", jsonBody)
	eventData := b.events[0].String()
	if event.String() != eventData {
		t.Errorf("Event data should match")
	}
}
