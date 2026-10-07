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
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

// TestResetFiltersOnSwitch verifies that screens for the current character or corporation
// clear their search and filters and reset modes to their defaults when that switches.
func TestResetFiltersOnSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip(ui.SkipUITestReason)
	}
	db, st, factory := testutil.NewDBOnDisk(t)
	defer db.Close()
	newUI := func(t *testing.T, isMobile bool) baseUI {
		return testdouble.NewUIFake(testdouble.UIParams{
			App:      test.NewTempApp(t),
			IsMobile: isMobile,
			Storage:  st,
		})
	}
	switchCharacter := func(t *testing.T, u baseUI) {
		u.Signals().CurrentCharacterExchanged.Emit(t.Context(), factory.CreateCharacter())
	}
	switchCorporation := func(t *testing.T, u baseUI) {
		u.Signals().CurrentCorporationExchanged.Emit(t.Context(), factory.CreateCorporation())
	}

	for _, isMobile := range []bool{true, false} {
		t.Run(fmt.Sprintf("wallet transactions of character mobile=%v", isMobile), func(t *testing.T) {
			u := newUI(t, isMobile)
			a := NewCharacterWalletTransaction(u)
			a.searchEntry.SetText("trit")
			if isMobile {
				a.filterChip.SetOptions(xwidget.NewFilterOptionMultiChoice(walletTransactionFilterActivity, []string{marketTransactionActivityBuy}))
				a.filterChip.SetSelected(map[string]string{walletTransactionFilterActivity: marketTransactionActivityBuy})
			} else {
				a.selectActivity.SetSelected(marketTransactionActivityBuy)
			}

			switchCharacter(t, u)

			assert.Empty(t, a.searchEntry.Text)
			assert.Empty(t, a.currentFilter())
		})
		t.Run(fmt.Sprintf("wallet transactions of corporation mobile=%v", isMobile), func(t *testing.T) {
			u := newUI(t, isMobile)
			a := NewCorporationWalletTransactions(u, app.Division1)
			a.searchEntry.SetText("trit")
			if isMobile {
				a.filterChip.SetOptions(xwidget.NewFilterOptionMultiChoice(walletTransactionFilterActivity, []string{marketTransactionActivityBuy}))
				a.filterChip.SetSelected(map[string]string{walletTransactionFilterActivity: marketTransactionActivityBuy})
			} else {
				a.selectActivity.SetSelected(marketTransactionActivityBuy)
			}

			switchCorporation(t, u)

			assert.Empty(t, a.searchEntry.Text)
			assert.Empty(t, a.currentFilter())
		})
		t.Run(fmt.Sprintf("contracts of corporation mobile=%v", isMobile), func(t *testing.T) {
			u := newUI(t, isMobile)
			a := NewContractsForCorporation(u)
			a.searchEntry.SetText("ship")
			a.selectStatus.SetSelected(contractStatusHistory)
			if isMobile {
				a.filterChip.SetOptions(xwidget.NewFilterOptionMultiChoice(contractsFilterType, []string{"Courier"}))
				a.filterChip.SetSelected(map[string]string{contractsFilterType: "Courier"})
			} else {
				a.selectType.SetOptions([]string{"Courier"})
				a.selectType.SetSelected("Courier")
			}

			switchCorporation(t, u)

			assert.Empty(t, a.searchEntry.Text)
			assert.Equal(t, contractsFilter{status: contractStatusAllActive}, a.currentFilter())
		})
		t.Run(fmt.Sprintf("skill catalogue mobile=%v", isMobile), func(t *testing.T) {
			u := newUI(t, isMobile)
			a := NewSkillCatalogue(u)
			a.searchEntry.SetText("gun")
			a.selectMain.SetSelected(skillCatalogueMySkill)
			if isMobile {
				a.filterChip.SetOptions(xwidget.NewFilterOptionMultiChoice(skillCatalogueFilterGroup, []string{"Gunnery"}))
				a.filterChip.SetSelected(map[string]string{skillCatalogueFilterGroup: "Gunnery"})
			} else {
				a.selectGroup.SetOptions([]string{"Gunnery"})
				a.selectGroup.SetSelected("Gunnery")
			}

			switchCharacter(t, u)

			assert.Empty(t, a.searchEntry.Text)
			assert.Equal(t, skillCatalogueFilter{main: skillCatalogueAllSkill}, a.currentFilter())
		})
		t.Run(fmt.Sprintf("asset browser of character mobile=%v", isMobile), func(t *testing.T) {
			u := newUI(t, isMobile)
			a := NewCharacterBrowser(u)
			a.Navigation.searchEntry.SetText("jita")
			a.Navigation.selectCategory.SetSelected(categoryAll)

			switchCharacter(t, u)

			assert.Empty(t, a.Navigation.searchEntry.Text)
			assert.Equal(t, categoryPersonal, a.Navigation.selectCategory.Selected)
		})
		t.Run(fmt.Sprintf("asset browser of corporation mobile=%v", isMobile), func(t *testing.T) {
			u := newUI(t, isMobile)
			a := NewCorporationBrowser(u)
			a.Navigation.searchEntry.SetText("jita")
			a.Navigation.selectCategory.SetSelected(categoryAll)

			switchCorporation(t, u)

			assert.Empty(t, a.Navigation.searchEntry.Text)
			assert.Equal(t, categoryOffice, a.Navigation.selectCategory.Selected)
		})
	}
	t.Run("wallet journal of character", func(t *testing.T) {
		u := newUI(t, false)
		a := NewCharacterWalletJournal(u)
		a.selectType.SetOptions([]string{"Bounty"})
		a.selectType.SetSelected("Bounty")

		switchCharacter(t, u)

		assert.Empty(t, a.selectType.Selected)
	})
	t.Run("wallet journal of corporation", func(t *testing.T) {
		u := newUI(t, false)
		a := NewCorporationWalletJournal(u, app.Division1)
		a.selectType.SetOptions([]string{"Bounty"})
		a.selectType.SetSelected("Bounty")

		switchCorporation(t, u)

		assert.Empty(t, a.selectType.Selected)
	})
	t.Run("communications of character", func(t *testing.T) {
		u := newUI(t, true)
		a := NewCommunicationsForCharacter(u)
		a.MessagePane.set(app.NotificationGroups()[0]) // a folder is selected after the first load
		a.MessagePane.searchEntry.SetText("war")
		a.MessagePane.filterChip.SetOptions(xwidget.NewFilterOptionMultiChoice(
			communicationsFilterStatus,
			[]string{communicationsFilterStatusRead, communicationsFilterStatusUnread},
		))
		a.MessagePane.filterChip.SetSelected(map[string]string{communicationsFilterStatus: communicationsFilterStatusUnread})
		c := factory.CreateCharacter()
		factory.CreateCharacterSectionStatus(testutil.CharacterSectionStatusParams{
			CharacterID: c.ID,
			Section:     app.SectionCharacterNotifications,
			CompletedAt: time.Now().UTC(),
		}) // with data the screen keeps its folder, so only the switch can reset the filter

		u.Signals().CurrentCharacterExchanged.Emit(t.Context(), c)

		assert.Empty(t, a.MessagePane.searchEntry.Text)
		assert.False(t, a.MessagePane.filterChip.IsOn())
	})
}
