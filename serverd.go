package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/davidsbond/sse"
	"github.com/davidsbond/sse/broker"
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

func handleWebhookConnect(ctx *macaron.Context, events broker.Broker) {
	if handled := handleFacebookVerification(ctx); handled {
		return
	}

	channel := ctx.Params(":channel")
	go func() {
		for {
			time.Sleep(30 * time.Second)
			err := events.BroadcastTo(
				ctx.Params(":channel"),
				sse.NewEvent("ping", []byte("ping")),
			)
			if err != nil {
				fmt.Println("Disconnected")
				break
			}
		}
	}()
	ctx.Redirect(fmt.Sprintf("/events?id=%s", channel))
}

func handleWebhookForward(ctx *macaron.Context, events broker.Broker) {
	body, _ := ctx.Req.Body().Bytes()
	err := events.BroadcastTo(
		ctx.Params(":channel"),
		sse.NewEvent("webhook", body),
	)
	if err != nil {
		ctx.Status(400)
	} else {
		ctx.Status(200)
	}
}

func main() {
	events := sse.NewBroker(sse.Config{
		Timeout:      10 * time.Second,
		Tolerance:    3,
		ErrorHandler: nil,
	})

	m := macaron.Classic()
	m.Map(events)
	m.Use(macaron.Renderer())
	m.Get("/webhook/:channel", handleWebhookConnect)
	m.Post("/webhook/:channel", handleWebhookForward)

	host, port := macaron.GetDefaultListenInfo()
	addr := host + ":" + strconv.Itoa(port)
	mux := http.NewServeMux()
	mux.Handle("/", m)
	mux.HandleFunc("/events", events.ClientHandler)
	fmt.Println(http.ListenAndServe(addr, mux))
}
