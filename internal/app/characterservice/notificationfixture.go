package characterservice

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"math/rand/v2"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/icrowley/fake"

	"github.com/ErikKalkoken/evebuddy/internal/app"
)

// idKeyPattern matches ESI notification YAML field names holding entity IDs, e.g. charID, corpID, listOfCharIDs.
var idKeyPattern = regexp.MustCompile(`ID\d*s?$`)

// nameLinkPattern matches an embedded HTML entity link, e.g. `<a href="showinfo:1376//1001">Bruce Wayne</a>`.
var nameLinkPattern = regexp.MustCompile(`^(<a href="showinfo:(\d+)(?://\d+)?">).*(</a>)$`)

// ESI "showinfo" category codes used in name links.
const (
	linkTypeCharacter   = "1376"
	linkTypeCorporation = "2"
	linkTypeAlliance    = "16159"
)

type notificationFixture struct {
	NotificationID int64     `json:"notification_id"`
	Type           string    `json:"type"`
	SenderID       int64     `json:"sender_id"`
	SenderType     string    `json:"sender_type"`
	Timestamp      time.Time `json:"timestamp"`
	Text           string    `json:"text"`
	IsRead         bool      `json:"is_read"`
}

// WriteNotificationTypeFixtures writes the newest notification per distinct type found in
// storage as JSON, in the evenotification package's testdata shape, with IDs and names anonymized.
func (s *CharacterService) WriteNotificationTypeFixtures(ctx context.Context, writer io.Writer) error {
	notifications, err := s.ListAllNotifications(ctx)
	if err != nil {
		return err
	}
	seen := make(map[app.EveNotificationType]*app.CharacterNotification)
	for _, n := range notifications {
		if n.Type == app.UnknownNotification {
			continue
		}
		if cur, ok := seen[n.Type]; !ok || n.Timestamp.After(cur.Timestamp) {
			seen[n.Type] = n
		}
	}
	types := slices.Collect(maps.Keys(seen))
	slices.SortFunc(types, func(a, b app.EveNotificationType) int {
		return strings.Compare(a.String(), b.String())
	})
	fixtures := make([]notificationFixture, 0, len(types))
	for _, t := range types {
		n := seen[t]
		ids := make(map[int64]int64) // reset per entry so substitutions stay consistent within it

		text, _ := n.Text.Value()
		anonymizedText := text
		if text != "" {
			var data any
			if err := yaml.Unmarshal([]byte(text), &data); err != nil {
				return err
			}
			data = anonymize(data, ids)
			b, err := yaml.Marshal(data)
			if err != nil {
				return err
			}
			anonymizedText = string(b)
		}

		fixtures = append(fixtures, notificationFixture{
			NotificationID: n.NotificationID,
			Type:           n.Type.String(),
			SenderID:       anonymizeID(n.Sender.ID, ids),
			SenderType:     n.Sender.Category.String(),
			Timestamp:      n.Timestamp,
			Text:           anonymizedText,
			IsRead:         n.IsRead,
		})
	}
	b, err := json.MarshalIndent(fixtures, "", "    ")
	if err != nil {
		return err
	}
	_, err = writer.Write(b)
	return err
}

// anonymize recursively anonymizes a decoded YAML value: IDs under keys matching idKeyPattern
// via ids, and every other string via [anonymizeString].
func anonymize(v any, ids map[int64]int64) any {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if idKeyPattern.MatchString(k) {
				x[k] = anonymizeIDValue(val, ids)
			} else {
				x[k] = anonymize(val, ids)
			}
		}
		return x
	case []any:
		for i, val := range x {
			x[i] = anonymize(val, ids)
		}
		return x
	case string:
		return anonymizeString(x)
	default:
		return v
	}
}

// anonymizeIDValue anonymizes a value under an ID-like key: an integer ID (or numeric ID string), or a list of those.
func anonymizeIDValue(v any, ids map[int64]int64) any {
	switch x := v.(type) {
	case string:
		if id, err := strconv.ParseInt(x, 10, 64); err == nil {
			return strconv.FormatInt(anonymizeID(id, ids), 10)
		}
		return anonymizeString(x)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = anonymizeIDValue(item, ids)
		}
		return out
	default:
		if id, ok := toInt64(x); ok {
			return anonymizeID(id, ids)
		}
		return x
	}
}

// anonymizeString replaces a free-form YAML string with random placeholder content, preserving
// empty strings, the "showinfo" literal, and the href structure of embedded name links (only their
// display name is replaced), since those are required by evenotification's parsing/branching logic.
func anonymizeString(s string) string {
	if s == "" || s == "showinfo" {
		return s
	}
	if m := nameLinkPattern.FindStringSubmatch(s); m != nil {
		prefix, code, suffix := m[1], m[2], m[3]
		switch code {
		case linkTypeCharacter:
			return prefix + fake.FullName() + suffix
		case linkTypeCorporation, linkTypeAlliance:
			return prefix + fake.Company() + suffix
		default:
			return s
		}
	}
	n := min(max(len(strings.Fields(s)), 1), 10)
	return fake.WordsN(n)
}

// toInt64 converts the integer types produced by the YAML decoder (int64 or uint64) into an int64.
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case uint64:
		return int64(n), true
	default:
		return 0, false
	}
}

// anonymizeID returns a random placeholder for orig, reusing prior placeholders from ids.
func anonymizeID(orig int64, ids map[int64]int64) int64 {
	if v, ok := ids[orig]; ok {
		return v
	}
	v := rand.Int64N(900_000_000) + 100_000_000
	ids[orig] = v
	return v
}
