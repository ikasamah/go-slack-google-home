package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ikasamah/homecast"
	"github.com/nlopes/slack"
	"golang.org/x/sync/errgroup"
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

	if err := s.speak(ctx, body); err != nil {
		// Reload device, because address may have changed according to DHCP.
		log.Printf("[WARN] An error occurred in speak. Attempt to reload devices just now. err: %s", err)
		if err := s.addReaction(ev, "warning"); err != nil {
			return err
		}
		s.devices = homecast.LookupGoogleHome()
		if err := s.speak(ctx, body); err != nil {
			s.addReaction(ev, "no_entry_sign")
			return err
		}
	}
	return s.addReaction(ev, "sound")
}

func (s *SlackBot) speak(ctx context.Context, body string) error {
	var eg errgroup.Group
	for i := range s.devices {
		device := s.devices[i]
		eg.Go(func() error {
			log.Printf("[INFO] Attempting to make device speak: [%s]%s", device.AddrV4, device.Name)
			return device.Speak(ctx, body, s.lang)
		})
	}
	return eg.Wait()
}

func (s *SlackBot) addReaction(ev *slack.MessageEvent, emojiName string) error {
	msgRef := slack.NewRefToMessage(ev.Channel, ev.Timestamp)
	return s.client.AddReaction(emojiName, msgRef)
}
