package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/r3labs/sse"
	"gopkg.in/macaron.v1"
)

func handleFacebookVerification(ctx *macaron.Context) bool {
	modes := ctx.QueryStrings("hub.mode")
	if len(modes) != 1 || modes[0] != "subscribe" {
		return false
	}

	challenges := ctx.QueryStrings("hub.challenge")
	if len(challenges) != 1 {
		ctx.Status(400)
		return true
	}

	channel := ctx.Params(":channel")
	tokens := ctx.QueryStrings("hub.verify_token")
	if len(tokens) != 1 || tokens[0] != channel {
		ctx.Status(400)
		return true
	}

	ctx.PlainText(200, []byte(challenges[0]))
	return true
}

func handleWebhookConnect(ctx *macaron.Context, events *sse.Server) {
	if handled := handleFacebookVerification(ctx); handled {
		return
	}

	channel := ctx.Params(":channel")
	if !events.StreamExists(channel) {
		events.CreateStream(channel)
	}
	go func() {
		for {
			time.Sleep(10 * time.Second)
			events.Publish(channel, &sse.Event{
				Data: []byte("ping"),
			})
		}
	}()
	ctx.Redirect(fmt.Sprintf("/events?stream=%s", channel))
}

func handleWebhookForward(ctx *macaron.Context, events *sse.Server) {
	events.Publish(ctx.Params(":channel"), &sse.Event{
		Data: []byte("forwarding message"),
	})
	ctx.Status(200)
}

func main() {
	events := sse.New()
	events.AutoReplay = false

	m := macaron.Classic()
	m.Map(events)
	m.Use(macaron.Renderer())
	m.Get("/webhook/:channel", handleWebhookConnect)
	m.Post("/webhook/:channel", handleWebhookForward)

	host, port := macaron.GetDefaultListenInfo()
	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.Handle("/", m)
	mux.HandleFunc("/events", events.HTTPHandler)
	fmt.Println(http.ListenAndServe(addr, mux))
}
