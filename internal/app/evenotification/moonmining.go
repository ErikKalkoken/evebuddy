package evenotification

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/set"
	"github.com/antihax/goesi/notification"
	"github.com/dustin/go-humanize"
	"gopkg.in/yaml.v3"
)

type moonMiningInfo struct {
	moon *app.EveMoon
	text string
}

func makeMoonMiningBaseText(ctx context.Context, moonID int32, structureName string, eus *eveuniverseservice.EveUniverseService) (moonMiningInfo, error) {
	moon, err := eus.GetOrCreateMoonESI(ctx, moonID)
	if err != nil {
		return moonMiningInfo{}, err
	}
	text := fmt.Sprintf(
		"for **%s** at %s in %s",
		structureName,
		moon.Name,
		makeSolarSystemLink(moon.SolarSystem),
	)
	x := moonMiningInfo{
		moon: moon,
		text: text,
	}
	return x, nil
}

type oreItem struct {
	id     int32
	name   string
	volume float64
}

func makeOreText(ctx context.Context, ores map[int32]float64, eus *eveuniverseservice.EveUniverseService) (string, error) {
	ids := set.Collect(maps.Keys(ores))
	entities, err := eus.ToEntities(ctx, ids)
	if err != nil {
		return "", err
	}
	items := make([]oreItem, 0)
	for id, v := range ores {
		i := oreItem{
			id:     id,
			name:   entities[id].Name,
			volume: v,
		}
		items = append(items, i)
	}
	slices.SortFunc(items, func(a, b oreItem) int {
		return cmp.Compare(a.name, b.name)
	})
	lines := []string{"Estimated ore composition:"}
	for i := range slices.Values(items) {
		text := fmt.Sprintf("%s: %s m3", i.name, humanize.Comma(int64(i.volume)))
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n\n"), nil
}

// NEW

type moonminingAutomaticFracture struct {
	baseRenderer
}

func (n moonminingAutomaticFracture) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n moonminingAutomaticFracture) unmarshal(text string) (notification.MoonminingAutomaticFracture, setInt32, error) {
	var data notification.MoonminingAutomaticFracture
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Collect(maps.Keys(data.OreVolumeByType))
	return data, ids, nil
}
func (n moonminingAutomaticFracture) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	o, err := makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName, n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Extraction for %s has autofractured", data.StructureName)
	ores, err := makeOreText(ctx, data.OreVolumeByType, n.eus)
	if err != nil {
		return title, body, err
	}
	out := fmt.Sprintf("The extraction for %s "+
		"has reached the end of it's lifetime and has fractured automatically. The moon products are ready to be harvested.\n\n%s",
		o.text,
		ores,
	)
	body = out
	return title, body, nil
}

type moonminingExtractionStarted struct {
	baseRenderer
}

func (n moonminingExtractionStarted) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n moonminingExtractionStarted) unmarshal(text string) (notification.MoonminingExtractionStarted, setInt32, error) {
	var data notification.MoonminingExtractionStarted
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Collect(maps.Keys(data.OreVolumeByType))
	return data, ids, nil
}
func (n moonminingExtractionStarted) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	o, err := makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName, n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Extraction started at %s", data.StructureName)
	ores, err := makeOreText(ctx, data.OreVolumeByType, n.eus)
	if err != nil {
		return title, body, err
	}
	out := fmt.Sprintf("A moon mining extraction has been started %s.\n\n"+
		"The chunk will be ready on location at %s, "+
		"and will fracture automatically on %s.\n\n%s",
		o.text,
		fromLDAPTime(data.ReadyTime).Format(app.DateTimeFormat),
		fromLDAPTime(data.AutoTime).Format(app.DateTimeFormat),
		ores,
	)
	body = out
	return title, body, nil
}

type moonminingExtractionFinished struct {
	baseRenderer
}

func (n moonminingExtractionFinished) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n moonminingExtractionFinished) unmarshal(text string) (notification.MoonminingExtractionFinished, setInt32, error) {
	var data notification.MoonminingExtractionFinished
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Collect(maps.Keys(data.OreVolumeByType))
	return data, ids, nil
}
func (n moonminingExtractionFinished) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	o, err := makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName, n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Extraction finished at %s", data.StructureName)
	ores, err := makeOreText(ctx, data.OreVolumeByType, n.eus)
	if err != nil {
		return title, body, err
	}
	out := fmt.Sprintf(
		"The extraction %s is finished and the chunk is ready to be shot at.\n\n"+
			"The chunk will automatically fracture on %s.\n\n%s",
		o.text,
		fromLDAPTime(data.AutoTime).Format(app.DateTimeFormat),
		ores,
	)
	body = out
	return title, body, nil
}

type moonminingExtractionCancelled struct {
	baseRenderer
}

func (n moonminingExtractionCancelled) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n moonminingExtractionCancelled) unmarshal(text string) (notification.MoonminingExtractionCancelled, setInt32, error) {
	var data notification.MoonminingExtractionCancelled
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	var ids setInt32
	if data.CancelledBy != 0 {
		ids.Add(data.CancelledBy)
	}
	return data, ids, nil
}

func (n moonminingExtractionCancelled) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	var data notification.MoonminingExtractionCancelled
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return title, body, err
	}
	o, err := makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName, n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("Extraction canceled at %s", data.StructureName)
	cancelledBy := ""
	if data.CancelledBy != 0 {
		x, err := n.eus.GetOrCreateEntityESI(ctx, data.CancelledBy)
		if err != nil {
			return title, body, err
		}
		cancelledBy = fmt.Sprintf(" by %s", makeEveEntityProfileLink(x))
	}
	out := fmt.Sprintf(
		"An ongoing extraction for %s has been cancelled%s.",
		o.text,
		cancelledBy,
	)
	body = out
	return title, body, nil
}

type moonminingLaserFired struct {
	baseRenderer
}

func (n moonminingLaserFired) entityIDs(text string) (setInt32, error) {
	_, ids, err := n.unmarshal(text)
	if err != nil {
		return setInt32{}, err
	}
	return ids, nil
}

func (n moonminingLaserFired) unmarshal(text string) (notification.MoonminingLaserFired, setInt32, error) {
	var data notification.MoonminingLaserFired
	if err := yaml.Unmarshal([]byte(text), &data); err != nil {
		return data, setInt32{}, err
	}
	ids := set.Collect(maps.Keys(data.OreVolumeByType))
	if data.FiredBy != 0 {
		ids.Add(data.FiredBy)
	}
	return data, ids, nil
}
func (n moonminingLaserFired) render(ctx context.Context, text string, timestamp time.Time) (string, string, error) {
	var title, body string
	data, _, err := n.unmarshal(text)
	if err != nil {
		return title, body, err
	}
	o, err := makeMoonMiningBaseText(ctx, data.MoonID, data.StructureName, n.eus)
	if err != nil {
		return title, body, err
	}
	title = fmt.Sprintf("%s has fired it's moon drill", data.StructureName)
	firedBy := ""
	if data.FiredBy != 0 {
		x, err := n.eus.GetOrCreateEntityESI(ctx, data.FiredBy)
		if err != nil {
			return title, body, err
		}
		firedBy = fmt.Sprintf("by %s ", makeEveEntityProfileLink(x))
	}
	ores, err := makeOreText(ctx, data.OreVolumeByType, n.eus)
	if err != nil {
		return title, body, err
	}
	out := fmt.Sprintf(
		"The moon drill fitted to %s has been fired %sand the moon products are ready to be harvested.\n\n%s",
		o.text,
		firedBy,
		ores,
	)
	body = out
	return title, body, nil
}
