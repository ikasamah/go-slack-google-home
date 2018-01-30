package main

import (
	"context"
	"flag"

	"github.com/nlopes/slack"
)

func main() {
	token := flag.String("token", "", "API token for slack")
	lang := flag.String("lang", "en", "Language to speak")

	flag.Parse()

	ctx := context.Background()
	client := slack.New(*token)
	NewSlackBot(client, *lang).Run(ctx)
}
