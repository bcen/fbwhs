package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/davidsbond/sse"
	"github.com/davidsbond/sse/broker"
	"github.com/segmentio/ksuid"
	"gopkg.in/macaron.v1"
)

const (
	PingDelay          = 30 * time.Second
	SSEBrokerTimeout   = 10 * time.Second
	SSEBrokerTolerance = 3
	TheOHSHITLimit     = 500
)

type WebhookForwardResponse struct {
	Header http.Header `json:"header"`
	Body   string      `json:"body"`
}

type WebhookHandler struct {
	sync.Mutex
	subscriptions map[string][]string
	eventIDLookup map[string]string
}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		subscriptions: make(map[string][]string),
		eventIDLookup: make(map[string]string),
	}
}

func (w *WebhookHandler) Subscribe(webhookID string) (string, error) {
	w.Lock()
	defer w.Unlock()
	if len(w.eventIDLookup) >= TheOHSHITLimit {
		log.Printf("Exceeded TheOHSHITLimit: %d", TheOHSHITLimit)
		return "", fmt.Errorf("🤷")
	}

	eventID := ksuid.New().String()
	eventIDs := w.subscriptions[webhookID]
	eventIDs = append(eventIDs, eventID)
	w.subscriptions[webhookID] = eventIDs
	w.eventIDLookup[eventID] = webhookID
	return eventID, nil
}

func (w *WebhookHandler) Unsubscribe(eventID string) bool {
	w.Lock()
	defer w.Unlock()
	wid, ok := w.eventIDLookup[eventID]
	if !ok {
		return false
	}

	delete(w.eventIDLookup, eventID)
	eventIDs := w.subscriptions[wid]
	for i, id := range eventIDs {
		if eventID == id {
			eventIDs = append(eventIDs[:i], eventIDs[i+1:]...)
			w.subscriptions[wid] = eventIDs
			break
		}
	}
	return true
}

func (w *WebhookHandler) EventIDs(webhookID string) []string {
	return w.subscriptions[webhookID]
}

func handleFacebookVerification(ctx *macaron.Context) bool {
	modes := ctx.QueryStrings("hub.mode")
	if len(modes) != 1 || modes[0] != "subscribe" {
		return false
	}

	challenges := ctx.QueryStrings("hub.challenge")
	if len(challenges) != 1 {
		ctx.Status(http.StatusBadRequest)
		return true
	}

	wid := ctx.Params(":wid")
	tokens := ctx.QueryStrings("hub.verify_token")
	if len(tokens) != 1 || tokens[0] != wid {
		ctx.Status(http.StatusBadRequest)
		return true
	}

	ctx.PlainText(http.StatusOK, []byte(challenges[0]))
	return true
}

func handleWebhookConnect(ctx *macaron.Context, events broker.Broker, w *WebhookHandler) {
	if handled := handleFacebookVerification(ctx); handled {
		return
	}

	wid := ctx.Params(":wid")
	eventID, err := w.Subscribe(wid)
	if err != nil {
		ctx.PlainText(http.StatusBadRequest, []byte(err.Error()))
		return
	}

	go func() {
		for {
			time.Sleep(PingDelay)
			err := events.BroadcastTo(
				eventID,
				sse.NewEvent("ping", []byte("ping")),
			)
			if err != nil {
				w.Unsubscribe(eventID)
				log.Printf(
					"Disconnected, eventID: %s, webhookID: %s, %d consumer(s)"+
						" left\n",
					eventID, wid, len(w.EventIDs(wid)),
				)
				break
			}
		}
	}()

	ctx.Redirect(fmt.Sprintf("/events?id=%s", eventID))
}

func handleWebhookForward(ctx *macaron.Context, events broker.Broker, w *WebhookHandler) {
	wid := ctx.Params(":wid")
	eventIDs := w.EventIDs(wid)
	if len(eventIDs) == 0 {
		ctx.Status(http.StatusBadRequest)
		return
	}

	body, _ := ctx.Req.Body().String()
	resp := WebhookForwardResponse{Header: ctx.Req.Header, Body: body}
	b, err := json.Marshal(resp)
	if err != nil {
		ctx.PlainText(http.StatusBadRequest, []byte(err.Error()))
		return
	}

	for _, eventID := range eventIDs {
		events.BroadcastTo(eventID, sse.NewEvent("webhook", b))
	}

	ctx.Status(http.StatusOK)
}

func handleEvents(wh *WebhookHandler, events broker.Broker) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		eventID := r.URL.Query().Get("id")
		if _, ok := wh.eventIDLookup[eventID]; !ok {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		events.ClientHandler(rw, r)
	}
}

func main() {
	wh := NewWebhookHandler()
	events := sse.NewBroker(sse.Config{
		Timeout:   SSEBrokerTimeout,
		Tolerance: SSEBrokerTolerance,
	})

	m := macaron.Classic()
	m.Map(events)
	m.Map(wh)
	m.Use(macaron.Renderer())
	m.Get("/webhook/:wid", handleWebhookConnect)
	m.Post("/webhook/:wid", handleWebhookForward)

	host, port := macaron.GetDefaultListenInfo()
	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.Handle("/", m)
	mux.HandleFunc("/events", handleEvents(wh, events))
	log.Fatal(http.ListenAndServe(addr, mux))
}
