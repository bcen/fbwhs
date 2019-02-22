package main

import (
	"testing"
)

func TestSubscription(t *testing.T) {
	w := NewWebhookHandler()
	wid := "abc123"
	id1, _ := w.Subscribe(wid)
	id2, _ := w.Subscribe(wid)
	if id1 == id2 {
		t.Errorf("Each subscription should be unique")
	}
}

func TestUnsubscribe(t *testing.T) {
	w := NewWebhookHandler()
	wid := "abc123"
	eventID, _ := w.Subscribe(wid)
	if !w.Unsubscribe(eventID) {
		t.Errorf("Should return true if subscribed")
	}
}

func TestUnsubscribeDeleted(t *testing.T) {
	w := NewWebhookHandler()
	if w.Unsubscribe("deleted eventID") {
		t.Errorf("Should return false for deleted")
	}

	eventID := "abc123"
	w.Subscribe(eventID)
	w.Unsubscribe(eventID)
	if w.Unsubscribe(eventID) {
		t.Errorf("Should return false for deleted")
	}
}

func TestGetEventIDs(t *testing.T) {
	w := NewWebhookHandler()
	wid := "abc123"
	w.Subscribe(wid)
	if len(w.EventIDs(wid)) == 0 {
		t.Errorf("Should have subscriptions")
	}

	if len(w.EventIDs("foo")) != 0 {
		t.Errorf("Should not have subscriptions")
	}
}
