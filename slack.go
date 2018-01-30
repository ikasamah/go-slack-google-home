package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/nlopes/slack"
	"golang.org/x/sync/errgroup"
)

type SlackBot struct {
	client        *slack.Client
	lang          string
	connectedUser *slack.UserDetails
	devices       []*CastDevice
}

func NewSlackBot(client *slack.Client, lang string) *SlackBot {
	return &SlackBot{
		client: client,
		lang:   lang,
	}
}

func (s *SlackBot) Run(ctx context.Context) {
	rtm := s.client.NewRTM()

	go rtm.ManageConnection()

	// Handle slack events
	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.ConnectedEvent:
			s.connectedUser = ev.Info.User
			s.devices = LookupGoogleHome()
			log.Printf("[INFO] Connected: user_id=%s", s.connectedUser.ID)
		case *slack.MessageEvent:
			if err := s.handleMessageEvent(ctx, ev); err != nil {
				log.Printf("[ERROR] Failed to handle message: %s", err)
			}
		case *slack.InvalidAuthEvent:
			log.Print("[ERROR] Failed to auth")
			return
		}
	}
}

func (s *SlackBot) handleMessageEvent(ctx context.Context, ev *slack.MessageEvent) error {
	mention := fmt.Sprintf("<@%s> ", s.connectedUser.ID)
	if !strings.HasPrefix(ev.Msg.Text, mention) {
		return nil
	}
	body := strings.TrimPrefix(ev.Msg.Text, mention)

	var eg errgroup.Group
	for i := range s.devices {
		device := s.devices[i]
		eg.Go(func() error {
			return device.Speak(ctx, body, s.lang)
		})
	}
	return eg.Wait()
}
