package main

import (
	"gopkg.in/macaron.v1"
)

func handleVerification(ctx *macaron.Context) {
	modes := ctx.QueryStrings("hub.mode")
	if len(modes) != 1 || modes[0] != "subscribe" {
		ctx.Status(400)
		return
	}

	challenges := ctx.QueryStrings("hub.challenge")
	if len(challenges) != 1 {
		ctx.Status(400)
		return
	}

	channel := ctx.Params(":channel")
	tokens := ctx.QueryStrings("hub.verify_token")
	if len(tokens) != 1 || tokens[0] != channel {
		ctx.Status(400)
		return
	}

	ctx.PlainText(200, []byte(challenges[0]))
}

func main() {
	m := macaron.Classic()
	m.Use(macaron.Renderer())
	m.Get("/webhook/:channel", handleVerification)
	m.Run()
}
