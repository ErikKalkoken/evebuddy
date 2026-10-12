package charactermanager

import (
	"context"
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil/testdouble"
)

// uiFake adds the methods this package needs to the shared UI fake.
type uiFake struct {
	*testdouble.UIFake
}

func (u *uiFake) CurrentCharacter() *app.Character                 { return nil }
func (u *uiFake) CurrentCorporation() *app.Corporation             { return nil }
func (u *uiFake) LoadCharacter(_ context.Context, _ int64) error   { return nil }
func (u *uiFake) LoadCorporation(_ context.Context, _ int64) error { return nil }
func (u *uiFake) SetAnyCharacter(_ context.Context) error          { return nil }
func (u *uiFake) SetAnyCorporation(_ context.Context) error        { return nil }

func TestAdmin_DeleteCharacter(t *testing.T) {
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// setup creates a corporation with two characters, so it is not deleted with one of them,
	// and corporation data no remaining character has permission for.
	setup := func(t *testing.T) (*admin, *uiFake, adminRow, *app.CorporationIndustryJob) {
		testutil.MustTruncateTables(db)
		httpmock.Reset()
		u := &uiFake{testdouble.NewUIFake(testdouble.UIParams{App: test.NewTempApp(t), Storage: st})}
		t.Cleanup(u.Corporation().Stop)
		corporation := factory.CreateCorporation()
		ec1 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		c1 := factory.CreateCharacter(storage.CreateCharacterParams{ID: ec1.ID})
		ec2 := factory.CreateEveCharacter(storage.CreateEveCharacterParams{CorporationID: corporation.ID})
		factory.CreateCharacter(storage.CreateCharacterParams{ID: ec2.ID})
		j := factory.CreateCorporationIndustryJob(storage.UpdateOrCreateCorporationIndustryJobParams{
			CorporationID: corporation.ID,
		})
		factory.CreateCorporationSectionStatus(testutil.CorporationSectionStatusParams{
			CorporationID: corporation.ID,
			Section:       app.SectionCorporationIndustryJobs,
		})
		cw := newManageCharacters(u, test.NewTempWindow(t, nil))
		cw.stop() // the snackbar would render text concurrently with the test
		r := adminRow{
			characterID:   c1.ID,
			characterName: c1.EveCharacter.Name,
			corporationID: corporation.ID,
		}
		return cw.characterAdmin, u, r, j
	}
	t.Run("removes corporation data before CharacterRemoved", func(t *testing.T) {
		a, u, r, j := setup(t)
		removed := make(chan bool, 1)
		u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			_, err := st.GetCorporationIndustryJob(ctx, j.CorporationID, j.JobID)
			removed <- errors.Is(err, app.ErrNotFound)
		})
		// when
		a.deleteCharacter(context.Background(), r)
		// then
		select {
		case got := <-removed:
			assert.True(t, got, "corporation data still there when CharacterRemoved was emitted")
		case <-time.After(5 * time.Second):
			t.Fatal("CharacterRemoved not emitted")
		}
	})
}
