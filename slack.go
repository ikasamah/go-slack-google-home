package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/ikasamah/homecast"
	"github.com/nlopes/slack"
)

type SlackBot struct {
	client        *slack.Client
	lang          string
	connectedUser *slack.UserDetails
	devices       []*homecast.CastDevice
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
			s.devices = homecast.LookupGoogleHome()
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

	var mErr *multierror.Error
	for i := range s.devices {
		go func(i int) {
			device := s.devices[i]
			log.Printf("[INFO] Attempting to make device speak: [%s]%s", device.AddrV4, device.Name)
			if err := device.Speak(ctx, body, s.lang); err != nil {
				log.Printf("[ERROR] Failed to make device speak: %s", err)
				mErr = multierror.Append(mErr, err)
			}
		}(i)
	}

	if mErr != nil {
		return mErr
	}
	msgRef := slack.NewRefToMessage(ev.Channel, ev.Timestamp)
	return s.client.AddReaction("sound", msgRef)
}
