package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/set"
)

type notification struct {
	NotificationID int       `json:"notification_id"`
	Text           string    `json:"text"`
	Timestamp      time.Time `json:"timestamp"`
	Type           string    `json:"type"`
}

type myEUS struct{}

func (s *myEUS) GetOrCreateEntityESI(ctx context.Context, id int32) (*app.EveEntity, error) {
	o := &app.EveEntity{
		ID:       id,
		Name:     "Name",
		Category: app.EveEntityUnknown,
	}
	return o, nil
}

func (s *myEUS) GetOrCreateLocationESI(ctx context.Context, id int64) (*app.EveLocation, error) {
	o := &app.EveLocation{
		ID:        app.UnknownLocationID,
		Name:      "Location",
		UpdatedAt: time.Now(),
	}
	return o, nil
}

func (s *myEUS) GetOrCreateMoonESI(ctx context.Context, id int32) (*app.EveMoon, error) {
	ss, nil := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	o := &app.EveMoon{
		ID:          id,
		Name:        "Moon",
		SolarSystem: ss,
	}
	return o, nil
}

func (s *myEUS) GetOrCreatePlanetESI(ctx context.Context, id int32) (*app.EvePlanet, error) {
	ss, _ := s.GetOrCreateSolarSystemESI(ctx, 30002537)
	et, _ := s.GetOrCreateTypeESI(ctx, 5)
	o := &app.EvePlanet{
		ID:          id,
		Name:        "Planet",
		SolarSystem: ss,
		Type:        et,
	}
	return o, nil
}

func (s *myEUS) GetOrCreateSolarSystemESI(ctx context.Context, id int32) (*app.EveSolarSystem, error) {
	o := &app.EveSolarSystem{
		ID:   id,
		Name: "System",
		Constellation: &app.EveConstellation{
			ID:   20000372,
			Name: "Constellation",
			Region: &app.EveRegion{
				ID:   10000030,
				Name: "Region",
			},
		},
	}
	return o, nil
}

func (s *myEUS) GetOrCreateTypeESI(ctx context.Context, id int32) (*app.EveType, error) {
	o := &app.EveType{
		ID:   id,
		Name: "Type",
		Group: &app.EveGroup{
			ID:   420,
			Name: "Group",
			Category: &app.EveCategory{
				ID:   5,
				Name: "Category",
			},
		},
	}
	return o, nil
}

func (s *myEUS) ToEntities(ctx context.Context, ids set.Set[int32]) (map[int32]*app.EveEntity, error) {
	m := make(map[int32]*app.EveEntity)
	for id := range ids.All() {
		o, _ := s.GetOrCreateEntityESI(ctx, id)
		m[id] = o
	}
	return m, nil
}

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatal("usage: gennotif {notifTypeName}")
	}

	data, err := os.ReadFile("/home/erik997/go/evebuddy-project/evebuddy/internal/app/evenotification/testdata/notifications.json")
	if err != nil {
		log.Fatal(err)
	}
	notifications := make([]notification, 0)
	if err := json.Unmarshal(data, &notifications); err != nil {
		log.Fatal(err)
	}
	m := make(map[string]notification)
	for _, n := range notifications {
		m[n.Type] = n
	}
	input := os.Args[1]
	nt, found := m[input]
	if !found {
		log.Fatal("notification type not found: " + input)
	}
	nt2, found := storage.EveNotificationTypeFromESIString(nt.Type)
	if !found {
		log.Fatal("notification type not found: " + input)
	}
	en := evenotification.New(&myEUS{})
	title, body, err := en.RenderESI(context.Background(), nt2, nt.Text, time.Now())
	if errors.Is(err, app.ErrNotFound) {
		log.Fatal("Notification type not supported")
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(title)
	fmt.Println("--------------------------------------------------")
	fmt.Println(body)
}
