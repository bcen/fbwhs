package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/davidsbond/sse"
	"github.com/davidsbond/sse/broker"
	"github.com/segmentio/ksuid"
)

const (
	PingDelay          = 30 * time.Second
	SSEBrokerTimeout   = 10 * time.Second
	SSEBrokerTolerance = 3
	TheOHSHITLimit     = 500
)

type (
	Webhook struct {
		Header http.Header `json:"header"`
		Body   string      `json:"body"`
	}

	WebhookHandler struct {
		sync.Mutex
		subscriptions map[string][]string
		eventIDLookup map[string]string
		sseBroker     broker.Broker
	}
)

func NewDefaultWebhookHandler() *WebhookHandler {
	return NewWebhookHandler(sse.NewBroker(sse.Config{
		Timeout:   SSEBrokerTimeout,
		Tolerance: SSEBrokerTolerance,
	}))
}

func NewWebhookHandler(b broker.Broker) *WebhookHandler {
	return &WebhookHandler{
		subscriptions: make(map[string][]string),
		eventIDLookup: make(map[string]string),
		sseBroker:     b,
	}
}

func (wh *WebhookHandler) Subscribe(webhookID string) (string, error) {
	wh.Lock()
	defer wh.Unlock()
	if len(wh.eventIDLookup) >= TheOHSHITLimit {
		log.Printf("Exceeded TheOHSHITLimit: %d", TheOHSHITLimit)
		return "", fmt.Errorf("🤷")
	}

	eventID := ksuid.New().String()
	eventIDs := wh.subscriptions[webhookID]
	eventIDs = append(eventIDs, eventID)
	wh.subscriptions[webhookID] = eventIDs
	wh.eventIDLookup[eventID] = webhookID
	return eventID, nil
}

func (wh *WebhookHandler) Unsubscribe(eventID string) bool {
	wh.Lock()
	defer wh.Unlock()
	wid, ok := wh.eventIDLookup[eventID]
	if !ok {
		return false
	}

	delete(wh.eventIDLookup, eventID)
	eventIDs := wh.subscriptions[wid]
	for i, id := range eventIDs {
		if eventID == id {
			eventIDs = append(eventIDs[:i], eventIDs[i+1:]...)
			wh.subscriptions[wid] = eventIDs
			break
		}
	}
	return true
}

func (wh *WebhookHandler) EventIDs(webhookID string) []string {
	return wh.subscriptions[webhookID]
}

func (wh *WebhookHandler) Forward(webhookID string, header http.Header, body string) error {
	eventIDs := wh.EventIDs(webhookID)
	if len(eventIDs) == 0 {
		return fmt.Errorf("No webhook connected")
	}
	w := Webhook{Header: header, Body: body}
	b, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("Unable to encode webhook to json")
	}

	for _, eventID := range eventIDs {
		wh.sseBroker.BroadcastTo(eventID, sse.NewEvent("webhook", b))
	}
	return nil
}

func (wh *WebhookHandler) KeepAlive(eventID string) {
	for {
		time.Sleep(PingDelay)
		err := wh.sseBroker.BroadcastTo(
			eventID,
			sse.NewEvent("ping", []byte("ping")),
		)
		if err != nil {
			wid := wh.eventIDLookup[eventID]
			wh.Unsubscribe(eventID)
			log.Printf(
				"Disconnected, eventID: %s, %d consumer(s) left",
				eventID, len(wh.subscriptions[wid]),
			)
			break
		}
	}
}

func (wh *WebhookHandler) HandleEvents(rw http.ResponseWriter, r *http.Request) {
	eventID := r.URL.Query().Get("id")
	if _, ok := wh.eventIDLookup[eventID]; !ok {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	wh.sseBroker.ClientHandler(rw, r)
}
