package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/barnybug/go-cast"
	"github.com/barnybug/go-cast/controllers"
	"github.com/hashicorp/mdns"
)

const (
	googleCastServiceName = "_googlecast._tcp"
	googleHomeModelInfo   = "md=Google Home"
)

type CastDevice struct {
	serviceEntry *mdns.ServiceEntry
	client       *cast.Client
}

func (g *CastDevice) Speak(ctx context.Context, text, lang string) error {
	url, err := tts(text, lang)
	if err != nil {
		return err
	}
	return g.Play(ctx, url)
}

func (g *CastDevice) Play(ctx context.Context, url *url.URL) error {
	if err := g.client.Connect(ctx); err != nil {
		return err
	}
	defer g.client.Close()

	media, err := g.client.Media(ctx)
	if err != nil {
		return err
	}
	mediaItem := controllers.MediaItem{
		ContentId:   url.String(),
		ContentType: "audio/mp3",
		StreamType:  "BUFFERED",
	}

	log.Printf("[INFO] Load media: content_id=%s", mediaItem.ContentId)
	_, err = media.LoadMedia(ctx, mediaItem, 0, true, nil)
	return err
}

func LookupGoogleHome() []*CastDevice {
	entriesCh := make(chan *mdns.ServiceEntry, 4)

	results := make([]*CastDevice, 0, 4)
	go func() {
		for entry := range entriesCh {
			for _, field := range entry.InfoFields {
				if field == googleHomeModelInfo {
					results = append(results, &CastDevice{
						serviceEntry: entry,
						client:       cast.NewClient(entry.AddrV4, entry.Port),
					})
				}
			}
		}
	}()

	mdns.Lookup(googleCastServiceName, entriesCh)
	close(entriesCh)

	return results
}

// tts provides text-to-speech sound url.
// NOTE: it seems that it's unofficial
func tts(text, lang string) (*url.URL, error) {
	base := "https://translate.google.com/translate_tts?client=tw-ob&ie=UTF-8&q=%s&tl=%s"
	return url.Parse(fmt.Sprintf(base, url.QueryEscape(text), url.QueryEscape(lang)))
}
