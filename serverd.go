package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"fbwhs/internal"
	"gopkg.in/macaron.v1"
)

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

func handleWebhookConnect(ctx *macaron.Context, wh *internal.WebhookHandler) {
	if handled := handleFacebookVerification(ctx); handled {
		return
	}

	wid := ctx.Params(":wid")
	eventID, err := wh.Subscribe(wid)
	if err != nil {
		ctx.PlainText(http.StatusBadRequest, []byte(err.Error()))
		return
	}

	go wh.KeepAlive(eventID)
	ctx.Redirect(fmt.Sprintf("/events?id=%s", eventID))
}

func handleWebhookForward(ctx *macaron.Context, wh *internal.WebhookHandler) {
	wid := ctx.Params(":wid")
	body, _ := ctx.Req.Body().String()

	if err := wh.Forward(wid, ctx.Req.Header, body); err != nil {
		log.Printf("Forward error: %s", err.Error())
		ctx.PlainText(http.StatusBadRequest, []byte(err.Error()))
		return
	}
	ctx.Status(http.StatusOK)
}

func main() {
	host, port := macaron.GetDefaultListenInfo()
	addr := host + ":" + strconv.Itoa(port)

	m := macaron.Classic()
	wh := internal.NewDefaultWebhookHandler()
	mux := http.NewServeMux()

	m.Map(wh)
	m.Use(macaron.Renderer())
	m.Get("/webhook/:wid", handleWebhookConnect)
	m.Post("/webhook/:wid", handleWebhookForward)
	mux.Handle("/", m)
	mux.HandleFunc("/events", wh.HandleEvents)
	log.Fatal(http.ListenAndServe(addr, mux))
}
