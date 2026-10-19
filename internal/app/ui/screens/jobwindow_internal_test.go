package screens

import (
	"fmt"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestShowIndustryJobWindow(t *testing.T) {
	t.Run("can show job with all details", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCharacter()
		now := time.Now()
		r := industryJobRow{
			activity:           app.Invention,
			blueprintType:      &app.EntityShort{ID: 1, Name: "Blueprint"},
			completedCharacter: optional.New(owner),
			completedDate:      optional.New(now),
			endDate:            now.Add(-time.Hour),
			installer:          owner,
			jobID:              42,
			licensedRuns:       optional.New(10),
			location:           factory.CreateEveLocationStation().ToEveLocationShort(),
			owner:              owner,
			pauseDate:          optional.New(now.Add(-2 * time.Hour)),
			probability:        optional.New[float32](0.5),
			productType:        optional.New(&app.EntityShort{ID: 2, Name: "Product"}),
			runs:               10,
			startDate:          now.Add(-3 * time.Hour),
			status:             app.JobDelivered,
			successfulRuns:     optional.New[int64](5),
		}
		// when
		showIndustryJobWindow(u, r)
		// then
		_, created := u.GetOrCreateWindow(fmt.Sprintf("industryjob-%d-%d", owner.ID, r.jobID))
		assert.False(t, created)
	})
	t.Run("registers status refresh when window is shown", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCharacter()
		r := industryJobRow{
			activity:      app.Manufacturing,
			blueprintType: &app.EntityShort{ID: 1, Name: "Blueprint"},
			endDate:       time.Now().Add(time.Hour),
			installer:     owner,
			jobID:         42,
			location:      factory.CreateEveLocationStation().ToEveLocationShort(),
			owner:         owner,
			status:        app.JobActive,
		}
		// when
		showIndustryJobWindow(u, r)
		// then
		xassert.Equal(t, 1, u.Signals().RefreshTickerExpired.Len())
	})
	t.Run("removes status refresh when window is closed", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCharacter()
		r := industryJobRow{
			activity:      app.Manufacturing,
			blueprintType: &app.EntityShort{ID: 1, Name: "Blueprint"},
			endDate:       time.Now().Add(time.Hour),
			installer:     owner,
			jobID:         42,
			location:      factory.CreateEveLocationStation().ToEveLocationShort(),
			owner:         owner,
			status:        app.JobActive,
		}
		key := fmt.Sprintf("industryjob-%d-%d", owner.ID, r.jobID)
		showIndustryJobWindow(u, r)
		w, _ := u.GetOrCreateWindow(key)
		// when
		w.Close()
		// then
		xassert.Equal(t, 0, u.Signals().RefreshTickerExpired.Len())
		_, created := u.GetOrCreateWindow(key)
		assert.True(t, created)
	})
	t.Run("reuses existing window", func(t *testing.T) {
		// given
		db, st, factory := testutil.NewDBOnDisk(t)
		defer db.Close()
		u := testdouble.NewUIFake(testdouble.UIParams{
			App:     test.NewTempApp(t),
			Storage: st,
		})
		owner := factory.CreateEveEntityCharacter()
		r := industryJobRow{jobID: 42, owner: owner}
		w, _ := u.GetOrCreateWindow(fmt.Sprintf("industryjob-%d-%d", owner.ID, r.jobID))
		// when
		showIndustryJobWindow(u, r)
		// then
		assert.Nil(t, w.Content())
		xassert.Equal(t, 0, u.Signals().RefreshTickerExpired.Len())
	})
}
