package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/testutil"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/xassert"
)

func TestEveEntityListItem(t *testing.T) {
	test.NewTempApp(t)

	t.Run("shows name and loads icon for entity", func(t *testing.T) {
		// given
		var loaded []*app.EveEntity
		var sizes []int
		w := NewEveEntityListItem(func(o *app.EveEntity, size int, setter func(fyne.Resource)) {
			loaded = append(loaded, o)
			sizes = append(sizes, size)
			setter(icons.Characterplaceholder64Jpeg)
		})
		ee := &app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter}
		// when
		w.Set(ee)
		// then
		xassert.Equal(t, "Bruce Wayne", w.nameLabel.Text)
		assert.False(t, w.nameLabel.TextStyle.Bold)
		xassert.Equal(t, []*app.EveEntity{ee}, loaded)
		xassert.Equal(t, []int{IconPixelSize}, sizes)
		xassert.Equal[fyne.Resource](t, icons.Characterplaceholder64Jpeg, w.iconImage.Resource)
	})
	t.Run("loads icon with custom pixel size", func(t *testing.T) {
		// given
		var sizes []int
		w := NewEveEntityListItem(func(_ *app.EveEntity, size int, _ func(fyne.Resource)) {
			sizes = append(sizes, size)
		})
		w.IconPixelSize = 32
		// when
		w.Set(&app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		// then
		xassert.Equal(t, []int{32}, sizes)
	})
	t.Run("shows name in bold without icon when ID is zero", func(t *testing.T) {
		// given
		var loaded []*app.EveEntity
		w := NewEveEntityListItem(func(o *app.EveEntity, _ int, _ func(fyne.Resource)) {
			loaded = append(loaded, o)
		})
		// when
		w.Set(&app.EveEntity{Name: "Total"})
		// then
		xassert.Equal(t, "Total", w.nameLabel.Text)
		assert.True(t, w.nameLabel.TextStyle.Bold)
		assert.Empty(t, loaded)
		xassert.Equal[fyne.Resource](t, icons.BlankSvg, w.iconImage.Resource)
	})
	t.Run("shows placeholder without icon for nil", func(t *testing.T) {
		// given
		var loaded []*app.EveEntity
		w := NewEveEntityListItem(func(o *app.EveEntity, _ int, _ func(fyne.Resource)) {
			loaded = append(loaded, o)
		})
		w.Set(&app.EveEntity{Name: "Total"})
		// when
		w.Set(nil)
		// then
		xassert.Equal(t, "?", w.nameLabel.Text)
		assert.False(t, w.nameLabel.TextStyle.Bold)
		assert.Empty(t, loaded)
		xassert.Equal[fyne.Resource](t, icons.BlankSvg, w.iconImage.Resource)
	})
	t.Run("shows entity from its parts", func(t *testing.T) {
		// given
		var loaded []*app.EveEntity
		w := NewEveEntityListItem(func(o *app.EveEntity, _ int, _ func(fyne.Resource)) {
			loaded = append(loaded, o)
		})
		// when
		w.Set2(42, "Wayne Enterprises", app.EveEntityCorporation)
		// then
		xassert.Equal(t, "Wayne Enterprises", w.nameLabel.Text)
		xassert.Equal(t, []*app.EveEntity{{ID: 42, Name: "Wayne Enterprises", Category: app.EveEntityCorporation}}, loaded)
	})
	t.Run("ignores icon loaded for previous entity", func(t *testing.T) {
		// given
		setters := make(map[int64]func(fyne.Resource))
		w := NewEveEntityListItem(func(o *app.EveEntity, _ int, setter func(fyne.Resource)) {
			setters[o.ID] = setter
		})
		w.Set(&app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		w.Set(&app.EveEntity{ID: 2, Name: "Clark Kent", Category: app.EveEntityCharacter})
		setters[2](icons.Characterplaceholder64Jpeg)
		// when
		setters[1](icons.Corporationplaceholder64Png)
		// then
		xassert.Equal[fyne.Resource](t, icons.Characterplaceholder64Jpeg, w.iconImage.Resource)
	})
	t.Run("ignores icon loaded for entity with same ID but other category", func(t *testing.T) {
		// given
		setters := make(map[app.EveEntityCategory]func(fyne.Resource))
		w := NewEveEntityListItem(func(o *app.EveEntity, _ int, setter func(fyne.Resource)) {
			setters[o.Category] = setter
		})
		w.Set(&app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		w.Set(&app.EveEntity{ID: 1, Name: "Wayne Enterprises", Category: app.EveEntityCorporation})
		setters[app.EveEntityCorporation](icons.Corporationplaceholder64Png)
		// when
		setters[app.EveEntityCharacter](icons.Characterplaceholder64Jpeg)
		// then
		xassert.Equal[fyne.Resource](t, icons.Corporationplaceholder64Png, w.iconImage.Resource)
	})
	t.Run("keeps blank icon when previous load finishes after totals row", func(t *testing.T) {
		// given
		var setter func(fyne.Resource)
		w := NewEveEntityListItem(func(_ *app.EveEntity, _ int, s func(fyne.Resource)) {
			setter = s
		})
		w.Set(&app.EveEntity{ID: 1, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		w.Set(&app.EveEntity{Name: "Total"})
		// when
		setter(icons.Characterplaceholder64Jpeg)
		// then
		xassert.Equal[fyne.Resource](t, icons.BlankSvg, w.iconImage.Resource)
	})
	t.Run("shows round icon for avatars", func(t *testing.T) {
		// given
		w := NewEveEntityListItem(func(_ *app.EveEntity, _ int, _ func(fyne.Resource)) {})
		w.IsAvatar = true
		// when
		w.Set(&app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		// then
		xassert.Equal(t, float32(IconUnitSize/2), w.iconImage.CornerRadius)
	})
	t.Run("shows square icon when no longer avatar", func(t *testing.T) {
		// given
		w := NewEveEntityListItem(func(_ *app.EveEntity, _ int, _ func(fyne.Resource)) {})
		w.IsAvatar = true
		w.Set(&app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		w.IsAvatar = false
		// when
		w.Set(&app.EveEntity{ID: 43, Name: "Wayne Enterprises", Category: app.EveEntityCorporation})
		// then
		xassert.Equal(t, float32(0), w.iconImage.CornerRadius)
	})
	t.Run("applies avatar change on refresh", func(t *testing.T) {
		// given
		w := NewEveEntityListItem(func(_ *app.EveEntity, _ int, _ func(fyne.Resource)) {})
		test.NewWindow(w)
		w.Set(&app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		w.IsAvatar = true
		// when
		w.Refresh()
		// then
		xassert.Equal(t, float32(IconUnitSize/2), w.iconImage.CornerRadius)
	})
	t.Run("applies icon size change on refresh", func(t *testing.T) {
		// given
		w := NewEveEntityListItem(func(_ *app.EveEntity, _ int, _ func(fyne.Resource)) {})
		test.NewWindow(w)
		w.IsAvatar = true
		w.Set(&app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		w.IconUnitSize = 40
		// when
		w.Refresh()
		// then
		xassert.Equal(t, fyne.NewSquareSize(40), w.iconImage.MinSize())
		xassert.Equal(t, float32(20), w.iconImage.CornerRadius)
	})
	t.Run("applies truncation change on refresh", func(t *testing.T) {
		// given
		w := NewEveEntityListItem(func(_ *app.EveEntity, _ int, _ func(fyne.Resource)) {})
		test.NewWindow(w)
		w.Set(&app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter})
		w.Truncation = fyne.TextTruncateEllipsis
		// when
		w.Refresh()
		// then
		xassert.Equal(t, fyne.TextTruncateEllipsis, w.nameLabel.Truncation)
	})
}

func TestMakeEveEntityColumn(t *testing.T) {
	test.NewTempApp(t)

	type row struct {
		ee *app.EveEntity
	}
	getEntity := func(r row) *app.EveEntity {
		return r.ee
	}

	t.Run("uses default width", func(t *testing.T) {
		// when
		c := MakeEveEntityColumn(MakeEveEntityColumnParams[row]{
			EIS:       testutil.NewEveImageServiceStub(),
			GetEntity: getEntity,
			Label:     "Character",
		})
		// then
		xassert.Equal(t, "Character", c.Label)
		xassert.Equal(t, float32(220), c.Width)
	})
	t.Run("uses custom width", func(t *testing.T) {
		// when
		c := MakeEveEntityColumn(MakeEveEntityColumnParams[row]{
			EIS:       testutil.NewEveImageServiceStub(),
			GetEntity: getEntity,
			Width:     100,
		})
		// then
		xassert.Equal(t, float32(100), c.Width)
	})
	t.Run("panics without entity getter", func(t *testing.T) {
		assert.Panics(t, func() {
			MakeEveEntityColumn(MakeEveEntityColumnParams[row]{
				EIS: testutil.NewEveImageServiceStub(),
			})
		})
	})
	t.Run("panics without image service", func(t *testing.T) {
		assert.Panics(t, func() {
			MakeEveEntityColumn(MakeEveEntityColumnParams[row]{
				GetEntity: getEntity,
			})
		})
	})
	t.Run("creates avatar items", func(t *testing.T) {
		// given
		c := MakeEveEntityColumn(MakeEveEntityColumnParams[row]{
			EIS:       testutil.NewEveImageServiceStub(),
			GetEntity: getEntity,
			IsAvatar:  true,
		})
		// when
		co := c.Create()
		// then
		assert.True(t, co.(*EveEntityListItem).IsAvatar)
	})
	t.Run("updates item with entity of row", func(t *testing.T) {
		// given
		eis := testutil.NewEveImageServiceStub()
		c := MakeEveEntityColumn(MakeEveEntityColumnParams[row]{
			EIS:       eis,
			GetEntity: getEntity,
		})
		co := c.Create()
		// when
		c.Update(row{ee: &app.EveEntity{ID: 42, Name: "Bruce Wayne", Category: app.EveEntityCharacter}}, co)
		// then
		w := co.(*EveEntityListItem)
		xassert.Equal(t, "Bruce Wayne", w.nameLabel.Text)
		xassert.Equal(t, eis.Character, w.iconImage.Resource)
	})
	t.Run("sorts by name ignoring case", func(t *testing.T) {
		// given
		c := MakeEveEntityColumn(MakeEveEntityColumnParams[row]{
			EIS:       testutil.NewEveImageServiceStub(),
			GetEntity: getEntity,
		})
		alpha := row{ee: &app.EveEntity{ID: 1, Name: "alpha"}}
		bravo := row{ee: &app.EveEntity{ID: 2, Name: "Bravo"}}
		// when/then
		assert.Negative(t, c.Sort(alpha, bravo))
		assert.Positive(t, c.Sort(bravo, alpha))
		assert.Zero(t, c.Sort(alpha, row{ee: &app.EveEntity{ID: 3, Name: "ALPHA"}}))
	})
}
