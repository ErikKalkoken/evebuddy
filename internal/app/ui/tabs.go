package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
)

// NewTabItem returns a new tab item whose content keeps a gap to the tab bar.
// Use it instead of [container.NewTabItem] for consistent spacing.
func NewTabItem(text string, content fyne.CanvasObject) *container.TabItem {
	return container.NewTabItem(text, PadTabContent(content))
}

// PadTabContent returns content with a gap to the tab bar.
// Use it when replacing the content of an existing tab item.
func PadTabContent(content fyne.CanvasObject) fyne.CanvasObject {
	return container.New(layout.NewCustomPaddedLayout(theme.Padding()/2, 0, 0, 0), content)
}
