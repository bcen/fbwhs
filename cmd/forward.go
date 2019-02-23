package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/r3labs/sse"
	"github.com/segmentio/ksuid"
)

const usage = `forward.

Usage:
  forward [options] <dest>

Options:
  -s -src        Webhook SSE source address. E.g. https://fbwhs.herokuapp.com/webhook/fb-callback
`

var src string

func init() {
	flag.StringVar(&src, "src", "", "Webhook SSE source")
	flag.StringVar(&src, "s", "", "Webhook SSE source")
}

type WebhookRequest struct {
	Header http.Header
	Body   string
}

func forwardEvent(msg *sse.Event, client *http.Client, dest string) {
	var req *http.Request
	eventType := string(msg.Event)

	if eventType == "ping" {
		// ignore ping event
		return
	} else if eventType != "webhook" {
		fmt.Println("Event not supported")
		return
	}

	fmt.Println("Forwarding event:")

	var wr WebhookRequest
	err := json.Unmarshal(msg.Data, &wr)
	if err != nil {
		fmt.Printf("Unable to decode json, error: %s\n", err.Error())
		return
	}
	fmt.Println(wr)

	req, _ = http.NewRequest("POST", dest, bytes.NewBufferString(wr.Body))
	req.Header = wr.Header
	resp, err := client.Do(req)

	if err != nil {
		fmt.Printf("Failed to forward event, error: %s\n", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("Error encountered when forwarding: %s\n", respBody)
		return
	}
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Error: <dest> is required")
		fmt.Println()
		fmt.Println(usage)
		os.Exit(1)
	}

	if src == "" {
		eventID := ksuid.New().String()
		src = fmt.Sprintf("https://fbwhs.herokuapp.com/webhook/%s", eventID)
	}

	dest := args[0]
	client := &http.Client{Timeout: 10 * time.Second}

	fmt.Printf(`Forwarding SSE from "%s" to "%s"`, src, dest)
	fmt.Printf("\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("curl -X POST -d 'test=123' \"%s\"\n", src)
	sse.NewClient(src).SubscribeRaw(func(msg *sse.Event) {
		go forwardEvent(msg, client, dest)
	})
}
