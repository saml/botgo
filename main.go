package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

type SlackUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RealTimeStart struct {
	WebSocketURL string      `json:"url"`
	Users        []SlackUser `json:"users"`
	Bots         []SlackUser `json:"bots"`
}

type SlackMessage struct {
	Type      string `json:"type"`
	UserID    string `json:"user"`
	ChannelID string `json:"channel"`
	Text      string `json:"text"`
}

func main() {
	var token string
	flag.StringVar(&token, "token", "", "Slack API token")
	flag.Parse()

	slackURL, err := url.Parse("https://slack.com/api/rtm.start")
	if err != nil {
		log.Fatal(err)
	}
	q := slackURL.Query()
	q.Set("token", token)
	slackURL.RawQuery = q.Encode()

	log.Print(slackURL.String())

	resp, err := http.Get(slackURL.String())
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var rtmStart RealTimeStart
	err = json.Unmarshal(body, &rtmStart)
	if err != nil {
		log.Fatal(err)
	}

	if rtmStart.WebSocketURL == "" {
		log.Fatal("Cannot retrieve websocket url")
	}
	log.Print(rtmStart.WebSocketURL)
	c, _, err := websocket.DefaultDialer.Dial(rtmStart.WebSocketURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)

		var msg SlackMessage
		for {
			err = c.ReadJSON(&msg)
			if err != nil {
				log.Print(err)
				return
			}
			log.Printf("> %s", msg)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	for {
		select {
		case <-interrupt:
			log.Print("interrupt")
			err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}
