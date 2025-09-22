package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage/testutil"
)

type notification struct {
	NotificationID int       `json:"notification_id"`
	Text           string    `json:"text"`
	Timestamp      time.Time `json:"timestamp"`
	Type           string    `json:"type"`
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	data, err := os.ReadFile("/home/erik997/go/evebuddy-project/evebuddy/internal/app/evenotification/testdata/notifications.json")
	if err != nil {
		log.Fatal(err)
	}
	notifications := make([]notification, 0)
	if err := json.Unmarshal(data, &notifications); err != nil {
		log.Fatal(err)
	}
	notifMap := make(map[string]notification)
	for _, n := range notifications {
		notifMap[n.Type] = n
	}
	args := flag.Args()
	if len(args) == 1 {
		input := args[0]
		nt, found := notifMap[input]
		if !found {
			log.Fatal("notification type not found: " + input)
		}
		err := printNotification(nt)
		if errors.Is(err, app.ErrNotFound) {
			log.Fatal("notification type not supported: " + nt.Type)
		} else if err != nil {
			log.Fatal(err)
		}
		return
	}
	fmt.Println("# EVE Buddy Notifications")
	fmt.Println()
	for _, nt := range notifMap {
		err := printNotification(nt)
		if err != nil && !errors.Is(err, app.ErrNotFound) {
			log.Fatal(err)
		}
	}
}

func printNotification(nt notification) error {
	nt2, found := storage.EveNotificationTypeFromESIString(nt.Type)
	if !found {
		return app.ErrNotFound
	}
	en := evenotification.New(&testutil.EveUniverseServiceFake{})
	title, body, err := en.RenderESI(context.Background(), nt2, nt.Text, time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("## %s\n\n", strings.Trim(nt.Type, " "))
	fmt.Printf("### %s\n\n", title)
	fmt.Println(body)
	fmt.Println()
	return nil
}
