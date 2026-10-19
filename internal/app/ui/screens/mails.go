package screens

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxwidget "github.com/ErikKalkoken/fyne-kx/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui/mailer"
	ihumanize "github.com/ErikKalkoken/evebuddy/internal/humanize"
	"github.com/ErikKalkoken/evebuddy/internal/icons"
	"github.com/ErikKalkoken/evebuddy/internal/singleinstance"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
	"github.com/ErikKalkoken/evebuddy/internal/xstrings"
	"github.com/ErikKalkoken/evebuddy/internal/xwidget"
)

type folderNodeCategory int

const (
	nodeCategoryBranch folderNodeCategory = iota + 1
	nodeCategoryLabel
	nodeCategoryList
)

type folderNodeType uint

const (
	folderNodeUndefined folderNodeType = iota
	folderNodeAll
	folderNodeAlliance
	folderNodeCorp
	folderNodeInbox
	folderNodeLabel
	folderNodeList
	folderNodeSent
	folderNodeTrash
)

// A mailFolderNode in the folder tree, e.g. the inbox
type mailFolderNode struct {
	Category    folderNodeCategory
	CharacterID int64
	IsLeaf      bool
	Type        folderNodeType
	Name        string
	ObjID       int64
	UnreadCount int
}

func (n *mailFolderNode) IsEmpty() bool {
	return n.CharacterID == 0
}

func (n *mailFolderNode) isBranch() bool {
	return n.Category == nodeCategoryBranch
}

func (n *mailFolderNode) icon() fyne.Resource {
	switch n.Type {
	case folderNodeInbox:
		return theme.DownloadIcon()
	case folderNodeSent:
		return theme.UploadIcon()
	case folderNodeTrash:
		return theme.DeleteIcon()
	}
	return theme.FolderIcon()
}

func (n mailFolderNode) UID() widget.TreeNodeID {
	return fmt.Sprintf("%d-%d-%d", n.CharacterID, n.Type, n.ObjID)
}

const mailsUpdated = "mailsUpdated"

type Mails struct {
	widget.BaseWidget

	OnUpdate       func(unread, missing int)
	MessagePane    *mailsMessagePane
	NavigationPane *mailsNavigationPane
	ReadingPane    *mailsReadingPane

	bodiesComplete    atomic.Bool // all mail bodies downloaded, so ticks can skip the check
	character         atomic.Pointer[app.Character]
	composeCharacters []app.EntityShort // senders offered for new mails in unified mode
	composeMenu       *fyne.Menu
	forCharacter      bool
	missingPercent    atomic.Int64
	sig               *singleinstance.Group
	u                 baseUI
	unreadCount       atomic.Int64
	updateRun         latestRun
}

func NewMailsForCharacter(u baseUI) *Mails {
	return newMails(u, true)
}

// NewUnifiedMails returns a mail screen showing the mails of all characters.
func NewUnifiedMails(u baseUI) *Mails {
	return newMails(u, false)
}

func newMails(u baseUI, forCharacter bool) *Mails {
	a := &Mails{
		forCharacter: forCharacter,
		u:            u,
		sig:          singleinstance.NewGroup(),
	}
	a.ExtendBaseWidget(a)
	a.MessagePane = newMailsMessagePane(a)
	a.NavigationPane = newMailsNavigationPane(a)
	a.ReadingPane = newMailsReadingPane(a)

	isMailSection := func(s app.CharacterSection) bool {
		switch s {
		case
			app.SectionCharacterMailLabels,
			app.SectionCharacterMailLists,
			app.SectionCharacterMailHeaders:
			return true
		}
		return false
	}
	if forCharacter {
		a.u.Signals().CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
			a.character.Store(c)
			fyne.Do(func() {
				a.ReadingPane.clear()
			})
			a.update(ctx)
		})
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if a.character.Load().IDOrZero() == arg.CharacterID && isMailSection(arg.Section) {
				a.update(ctx)
			}
		})
	} else {
		a.u.Signals().AppInit.AddListener(func(ctx context.Context, _ struct{}) {
			a.update(ctx)
		})
		a.u.Signals().CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
			if isMailSection(arg.Section) {
				a.update(ctx)
			}
		})
		a.u.Signals().CharacterAdded.AddListener(func(ctx context.Context, _ *app.Character) {
			a.update(ctx)
		})
		a.u.Signals().CharacterRemoved.AddListener(func(ctx context.Context, _ *app.EntityShort) {
			a.update(ctx)
		})
	}
	a.u.Signals().DataUpdated.AddListener(func(ctx context.Context, s string) {
		if s == mailsUpdated {
			a.NavigationPane.updateUnreadCounts(ctx)
			a.MessagePane.update(ctx)
			a.NavigationPane.updateDownloaded(ctx, true) // ticks may be skipping it
		}
	})
	a.u.Signals().RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		if a.bodiesComplete.Load() {
			return // new mails go through update, which checks again
		}
		a.NavigationPane.updateDownloaded(ctx, true)
	})
	return a
}

func (a *Mails) CreateRenderer() fyne.WidgetRenderer {
	// toolbar is not part of the reading pane, because mobile shows its own actions
	split1 := container.NewHSplit(
		a.MessagePane,
		container.NewBorder(a.ReadingPane.toolbar, nil, nil, nil, a.ReadingPane),
	)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(a.NavigationPane, split1)
	split2.SetOffset(0.15)
	p := theme.Padding()
	c := container.New(layout.NewCustomPaddedLayout(-p, 0, 0, 0), split2)
	return widget.NewSimpleRenderer(c)
}

func (a *Mails) update(ctx context.Context) {
	// discard running checks and keep ticks checking, also when returning early
	a.NavigationPane.downloadedRun.start()
	a.bodiesComplete.Store(false)
	isLatest := a.updateRun.start()
	clearAll := func() {
		if !isLatest() {
			return
		}
		// stored now, not queued, so a newer update's state is not overwritten
		a.unreadCount.Store(0)
		a.missingPercent.Store(0)
		a.MessagePane.currentFolder.Store(nil)
		a.MessagePane.updateRun.start() // discards message pane updates still loading
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.NavigationPane.clear()
			a.MessagePane.clear()
			a.ReadingPane.clear()
			a.callOnUpdate()
		})
	}
	setInfo := func(s string, i widget.Importance) {
		clearAll()
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.NavigationPane.composeButton.Disable()
			a.MessagePane.setInfo(s, i)
		})
	}
	setWaiting := func(characters int) {
		clearAll()
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.NavigationPane.composeButton.Enable() // sending does not need mail data
			a.MessagePane.setInfo("Waiting for mail data to be loaded...", widget.WarningImportance)
			if characters > 0 {
				a.NavigationPane.setCharacters(0, characters)
				a.callOnUpdate() // clearAll reported before the count was set
			}
		})
	}
	setError := func(err error) {
		clearAll() // so no outdated mails are shown with the error
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.NavigationPane.composeButton.Disable()
			a.MessagePane.setError(err)
		})
	}
	var characterID int64
	if a.forCharacter {
		characterID = a.character.Load().IDOrZero()
		if characterID == 0 {
			setInfo("No character", widget.LowImportance)
			return
		}
		hasData, err := a.u.Character().HasSection(ctx, characterID, app.SectionCharacterMailHeaders)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to build mail tree", "character", characterID, "error", err)
			setError(err)
			return
		}
		if !hasData {
			setWaiting(0)
			return
		}
	} else {
		names, err := a.u.Character().CharacterNames(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to build mail tree", "error", err)
			setError(err)
			return
		}
		var characters []app.EntityShort
		for id, name := range names {
			characters = append(characters, app.EntityShort{ID: id, Name: name})
		}
		slices.SortFunc(characters, func(a, b app.EntityShort) int {
			return strings.Compare(a.Name, b.Name)
		})
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.composeCharacters = characters
			a.composeMenu = a.makeComposeMenu()
		})
		if len(characters) == 0 {
			setInfo("No characters", widget.LowImportance)
			return
		}
		loaded, err := a.u.Character().CountCharactersWithSection(ctx, app.SectionCharacterMailHeaders)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("Failed to build mail tree", "error", err)
			setError(err)
			return
		}
		if loaded == 0 {
			setWaiting(len(characters))
			return
		}
		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.NavigationPane.setCharacters(loaded, len(characters))
		})
	}
	td, inbox, err := a.NavigationPane.fetchFolders(ctx, characterID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to build mail tree", "character", characterID, "error", err)
		setError(err)
		return
	}
	unread, err := a.NavigationPane.updateCountsInTree(ctx, characterID, td)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to update mail counts", "character", characterID, "error", err)
	}
	// keep showing the current folder if it still exists, e.g. after new mail arrived
	current := a.MessagePane.currentFolder.Load()
	folder := inbox
	if current != nil {
		if n, ok := td.Node(current.UID()); ok {
			folder = n
		}
	}
	isSameFolder := current != nil && current.UID() == folder.UID()
	if !isLatest() {
		return
	}
	a.MessagePane.currentFolder.Store(folder) // before set, so re-selecting the node is a no-op
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.NavigationPane.set(td, folder)
	})
	if isSameFolder {
		a.MessagePane.update(ctx)
	} else {
		a.MessagePane.setCurrentFolder(ctx, folder)
	}
	if !isLatest() {
		return
	}
	a.unreadCount.Store(int64(unread))
	a.NavigationPane.updateDownloaded(ctx, false) // OnUpdate is called below
	fyne.Do(func() {
		if !isLatest() {
			return
		}
		a.callOnUpdate()
	})
}

// mailsChanged refreshes everything that shows mails, including the other mail screens.
func (a *Mails) mailsChanged(ctx context.Context, characterID int64) {
	a.u.Signals().DataUpdated.Emit(ctx, mailsUpdated)        // returns after all mail screens are refreshed
	go a.u.Signals().CharacterChanged.Emit(ctx, characterID) // update character overview
	a.u.UpdateMailIndicator(ctx)
}

// CharactersLoaded reports how many characters have mails loaded. Must be called on the main thread.
func (a *Mails) CharactersLoaded() (loaded, total int) {
	return a.NavigationPane.charactersLoaded, a.NavigationPane.charactersTotal
}

func (a *Mails) callOnUpdate() {
	if a.OnUpdate == nil {
		return
	}
	a.OnUpdate(int(a.unreadCount.Load()), int(a.missingPercent.Load()))
}

func (a *Mails) showMailerWindow(mode mailer.Mode, mail *app.CharacterMail) {
	if a.forCharacter {
		if c := a.character.Load(); c != nil {
			a.openMailerWindow(c, mode, mail)
		}
		return
	}
	if mail == nil {
		return
	}
	a.showMailerWindowForCharacter(mail.CharacterID, mode, mail) // the mail's own character sends replies
}

func (a *Mails) showMailerWindowForCharacter(characterID int64, mode mailer.Mode, mail *app.CharacterMail) {
	go func() {
		c, err := a.u.Character().GetCharacter(context.Background(), characterID)
		if err != nil {
			slog.Error("Failed to load character for mailer", "characterID", characterID, "error", err)
			a.u.DisplaySnackbar("ERROR: Failed to open mailer: " + a.u.ErrorDisplay(err))
			return
		}
		fyne.Do(func() {
			a.openMailerWindow(c, mode, mail)
		})
	}()
}

func (a *Mails) openMailerWindow(c *app.Character, mode mailer.Mode, mail *app.CharacterMail) {
	w, err := mailer.NewWindow(a.u, c, mode, mail)
	if err != nil {
		ui.ShowErrorAndLog(
			"Failed to show mailer window",
			err,
			a.u.IsDeveloperMode(),
			a.u.MainWindow(),
		)
		return
	}
	w.Show()
}

func (a *Mails) MakeComposeMessageAction() (fyne.Resource, func()) {
	return theme.DocumentCreateIcon(), func() {
		a.showMailerWindow(mailer.New, nil)
	}
}

// Compose starts a new mail. In unified mode it first asks for the sending character
// with a menu shown below anchor, unless there is only one character.
func (a *Mails) Compose(anchor fyne.CanvasObject) {
	if a.forCharacter {
		a.showMailerWindow(mailer.New, nil)
		return
	}
	switch len(a.composeCharacters) {
	case 0:
		return
	case 1:
		a.showMailerWindowForCharacter(a.composeCharacters[0].ID, mailer.New, nil)
		return
	}
	xwidget.ShowPopUpMenuBelowLeading(anchor, a.composeMenu)
}

// makeComposeMenu returns the menu for choosing the sender of a new mail.
// It is built in advance, so the avatars can load before the menu is shown.
func (a *Mails) makeComposeMenu() *fyne.Menu {
	var items []*fyne.MenuItem
	for _, c := range a.composeCharacters {
		it := fyne.NewMenuItem(c.Name, func() {
			a.showMailerWindowForCharacter(c.ID, mailer.New, nil)
		})
		a.u.SetCharacterAvatarAsync(c.ID, func(r fyne.Resource) {
			it.Icon = r
		})
		items = append(items, it)
	}
	items = append(items, fyne.NewMenuItemSeparator(), fyne.NewMenuItem("Cancel", func() {})) // Fyne ignores taps on items without action
	return fyne.NewMenu("", items...)
}

type mailsNavigationPane struct {
	widget.BaseWidget

	charactersLabel       *widget.Label
	charactersLoaded      int
	charactersTotal       int
	composeButton         *widget.Button
	downloadedRun         latestRun
	folderDownloadedLabel *ttwidget.Label
	foldersTree           *xwidget.Tree[mailFolderNode]
	folderTotalLabel      *widget.Label
	ma                    *Mails
}

func newMailsNavigationPane(ma *Mails) *mailsNavigationPane {
	a := &mailsNavigationPane{
		charactersLabel:       ui.NewLabelWithTruncation("?"),
		folderDownloadedLabel: ttwidget.NewLabel(""),
		folderTotalLabel:      widget.NewLabel("?"),
		ma:                    ma,
	}
	a.ExtendBaseWidget(a)
	a.foldersTree = a.makeFolderTree()
	a.composeButton = widget.NewButtonWithIcon("Compose", theme.DocumentCreateIcon(), func() {
		ma.Compose(a.composeButton)
	})
	a.composeButton.Importance = widget.HighImportance
	a.composeButton.Disable()
	if ma.forCharacter {
		a.charactersLabel.Hide()
	}
	return a
}

func (a *mailsNavigationPane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewPadded(a.composeButton),
		container.New(
			layout.NewCustomPaddedVBoxLayout(0),
			a.charactersLabel,
			container.NewHBox(a.folderTotalLabel, layout.NewSpacer(), a.folderDownloadedLabel),
		),
		nil,
		nil,
		a.foldersTree,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *mailsNavigationPane) makeFolderTree() *xwidget.Tree[mailFolderNode] {
	t := xwidget.NewTree(
		func(_ bool) fyne.CanvasObject {
			return newMailFolderItemWidget()
		},
		func(n *mailFolderNode, _ bool, co fyne.CanvasObject) {
			co.(*mailFolderItemWidget).set(n)
		},
	)
	t.OnSelectedNode = func(n *mailFolderNode) {
		if n.isBranch() {
			t.UnselectAll()
			t.ToggleBranchNode(n)
			return
		}
		if c := a.ma.MessagePane.currentFolder.Load(); c != nil && c.UID() == n.UID() {
			return // already shown, e.g. re-selected after a refresh
		}
		go a.ma.MessagePane.setCurrentFolder(context.Background(), n)
	}
	return t
}

func (a *mailsNavigationPane) clear() {
	a.foldersTree.Clear()
	a.folderTotalLabel.SetText("?")
	a.folderDownloadedLabel.SetText("")
	a.charactersLabel.Text = "?"
	a.charactersLabel.Importance = widget.MediumImportance
	a.charactersLabel.Refresh()
	a.charactersLoaded = 0
	a.charactersTotal = 0
}

// setCharacters shows how many characters have mails loaded.
func (a *mailsNavigationPane) setCharacters(loaded, total int) {
	loaded = min(loaded, total) // the two counts can race with a deleted character
	a.charactersLoaded = loaded
	a.charactersTotal = total
	a.charactersLabel.Text = fmt.Sprintf("%d / %d characters", loaded, total)
	if loaded < total {
		a.charactersLabel.Importance = widget.WarningImportance
	} else {
		a.charactersLabel.Importance = widget.MediumImportance
	}
	a.charactersLabel.Refresh()
}

func (a *mailsNavigationPane) set(td *xwidget.TreeData[mailFolderNode], selected *mailFolderNode) {
	a.composeButton.Enable()
	var openBranches []widget.TreeNodeID
	a.foldersTree.Data().Walk(nil, func(n *mailFolderNode) bool {
		if a.foldersTree.IsBranchOpenNode(n) {
			openBranches = append(openBranches, n.UID())
		}
		return true
	})
	a.foldersTree.Set(td)
	for _, uid := range openBranches {
		if n, ok := td.Node(uid); ok {
			a.foldersTree.OpenBranchNode(n)
		}
	}
	a.foldersTree.SelectNode(selected)
}

// mailsDownloaded is the result of a check for downloaded mail bodies.
type mailsDownloaded struct {
	complete       bool
	downloaded     string
	hint           string
	missingPercent int
	total          string
}

// updateDownloaded shows how many mail bodies are downloaded.
// With notify it calls OnUpdate when the percentage changed.
func (a *mailsNavigationPane) updateDownloaded(ctx context.Context, notify bool) {
	isLatest := a.downloadedRun.start()
	r, ok := a.fetchDownloaded(ctx)
	if !ok {
		return
	}
	fyne.Do(func() {
		a.applyDownloaded(isLatest, r, notify)
	})
}

// applyDownloaded shows a download state unless a newer check was started.
// Must be called on the main thread.
func (a *mailsNavigationPane) applyDownloaded(isLatest func() bool, r mailsDownloaded, notify bool) {
	if !isLatest() {
		return
	}
	a.ma.bodiesComplete.Store(r.complete)
	changed := a.ma.missingPercent.Swap(int64(r.missingPercent)) != int64(r.missingPercent)
	a.folderTotalLabel.SetText(r.total)
	if r.downloaded == "" {
		a.folderDownloadedLabel.Hide()
	} else {
		a.folderDownloadedLabel.SetText(r.downloaded)
		a.folderDownloadedLabel.SetToolTip(r.hint)
		a.folderDownloadedLabel.Show()
	}
	if notify && changed {
		a.ma.callOnUpdate()
	}
}

// fetchDownloaded returns the current download state. It reports false when canceled.
func (a *mailsNavigationPane) fetchDownloaded(ctx context.Context) (mailsDownloaded, bool) {
	var r mailsDownloaded
	var total, missing int
	var err error
	if a.ma.forCharacter {
		characterID := a.ma.character.Load().IDOrZero()
		if characterID == 0 {
			return r, true
		}
		total, missing, err = a.ma.u.Character().CountMailsWithoutBody(ctx, characterID)
	} else {
		total, missing, err = a.ma.u.Character().CountAllMailsWithoutBody(ctx)
	}
	if app.IsContextCanceled(ctx, err) {
		return r, false
	}
	if err != nil {
		slog.Error("updateDownloaded", "error", err)
		r.total = "ERROR"
		return r, true
	}
	missing = min(missing, total) // the two counts can race with incoming mails
	r.complete = missing == 0
	p := message.NewPrinter(language.English)
	r.total = p.Sprintf("%d total", total)
	if total == 0 || missing == 0 {
		return r, true
	}
	r.missingPercent = (missing*100 + total - 1) / total // rounded up, so it never shows 100% while mails are missing
	r.downloaded = fmt.Sprintf("%d%%", 100-r.missingPercent)
	r.hint = p.Sprintf("%d / %d mails downloaded", total-missing, total)
	return r, true
}

// fetchFolders returns the folder tree of a character, or of all characters when characterID is 0.
func (a *mailsNavigationPane) fetchFolders(ctx context.Context, characterID int64) (*xwidget.TreeData[mailFolderNode], *mailFolderNode, error) {

	td := xwidget.NewTreeData[mailFolderNode]()

	// Add default folders
	var inbox *mailFolderNode
	defaultFolders := []struct {
		nodeType folderNodeType
		labelID  int64
		name     string
	}{
		{folderNodeInbox, app.MailLabelInbox, "Inbox"},
		{folderNodeSent, app.MailLabelSent, "Sent"},
		{folderNodeCorp, app.MailLabelCorp, "Corp"},
		{folderNodeAlliance, app.MailLabelAlliance, "Alliance"},
	}
	for _, o := range defaultFolders {
		n := &mailFolderNode{
			CharacterID: characterID,
			Category:    nodeCategoryLabel,
			Type:        o.nodeType,
			Name:        o.name,
			ObjID:       o.labelID,
		}
		if err := td.Add(nil, n, false); err != nil {
			return td, nil, err
		}
		if o.nodeType == folderNodeInbox {
			inbox = n
		}
	}

	// Add custom labels. They can not be merged, because each character has its own label IDs.
	var labels []*app.CharacterMailLabel
	if characterID != 0 {
		var err error
		labels, err = a.ma.u.Character().ListMailLabelsOrdered(ctx, characterID)
		if err != nil {
			return td, nil, err
		}
	}
	if len(labels) > 0 {
		n := &mailFolderNode{
			Category:    nodeCategoryBranch,
			CharacterID: characterID,
			Name:        "Labels",
			Type:        folderNodeLabel,
		}
		err := td.Add(nil, n, len(labels) > 0)
		if err != nil {
			return td, nil, err
		}
		for _, l := range labels {
			err := td.Add(n, &mailFolderNode{
				Category:    nodeCategoryLabel,
				CharacterID: characterID,
				Name:        l.Name.ValueOrZero(),
				ObjID:       l.LabelID,
				Type:        folderNodeLabel,
			}, false)
			if err != nil {
				return td, nil, err
			}
		}
	}

	// Add mailing lists
	var lists []*app.EveEntity
	var err error
	if characterID != 0 {
		lists, err = a.ma.u.Character().ListMailLists(ctx, characterID)
	} else {
		lists, err = a.ma.u.Character().ListAllMailLists(ctx)
	}
	if err != nil {
		return td, nil, err
	}
	if len(lists) > 0 {
		n := &mailFolderNode{
			Category:    nodeCategoryBranch,
			CharacterID: characterID,
			Name:        "Mailing Lists",
			Type:        folderNodeList,
		}
		err := td.Add(nil, n, len(lists) > 0)
		if err != nil {
			return td, nil, err
		}
		for _, l := range lists {
			err := td.Add(n, &mailFolderNode{
				Category:    nodeCategoryList,
				CharacterID: characterID,
				ObjID:       l.ID,
				Name:        l.Name,
				Type:        folderNodeList,
			}, false)
			if err != nil {
				return td, nil, err
			}
		}
	}
	// Add all folder
	err = td.Add(nil, &mailFolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeAll,
		Name:        "All",
		ObjID:       app.MailLabelAll,
	}, false)
	if err != nil {
		return td, nil, err
	}
	return td, inbox, nil
}

func (a *mailsNavigationPane) updateCountsInTree(ctx context.Context, characterID int64, td *xwidget.TreeData[mailFolderNode]) (int, error) {
	if td.IsEmpty() {
		return 0, nil
	}
	cs := a.ma.u.Character()
	var labelUnreadCounts, listUnreadCounts map[int64]int
	var unreadCount int
	var err error
	if characterID != 0 {
		labelUnreadCounts, err = cs.CountMailsUnreadByLabel(ctx, characterID)
		if err != nil {
			return 0, err
		}
		listUnreadCounts, err = cs.CountMailsUnreadByList(ctx, characterID)
		if err != nil {
			return 0, err
		}
		// summing label and list counts would count mails with several labels or lists repeatedly
		unreadCount, err = cs.CountMailsUnread(ctx, characterID)
		if err != nil {
			return 0, err
		}
	} else {
		labelUnreadCounts, err = cs.CountAllMailsUnreadByLabel(ctx)
		if err != nil {
			return 0, err
		}
		listUnreadCounts, err = cs.CountAllMailsUnreadByList(ctx)
		if err != nil {
			return 0, err
		}
		unreadCount, err = cs.CountAllMailsUnread(ctx)
		if err != nil {
			return 0, err
		}
	}

	var labelCount, listCount int
	for id, c := range labelUnreadCounts {
		if id > app.MailLabelAlliance {
			labelCount += c
		}
	}
	for _, c := range listUnreadCounts {
		listCount += c
	}

	td.Walk(nil, func(n *mailFolderNode) bool {
		var c int
		switch n.Type {
		case folderNodeAll:
			c = unreadCount
		case folderNodeInbox, folderNodeAlliance, folderNodeCorp:
			c = labelUnreadCounts[n.ObjID]
		case folderNodeLabel:
			if n.ObjID == 0 {
				c = labelCount
				break
			}
			c = labelUnreadCounts[n.ObjID]
		case folderNodeList:
			if n.ObjID == 0 {
				c = listCount
				break
			}
			c = listUnreadCounts[n.ObjID]
		}
		if n.UnreadCount != c {
			n.UnreadCount = c
		}
		return true
	})
	return unreadCount, nil
}

func (a *mailsNavigationPane) updateUnreadCounts(ctx context.Context) {
	td := a.foldersTree.Data()
	characterID := a.ma.character.Load().IDOrZero()
	unread, err := a.updateCountsInTree(ctx, characterID, td)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to update unread counts", "characterID", characterID, "error", err)
		return
	}
	a.ma.unreadCount.Store(int64(unread))
	fyne.Do(func() {
		a.foldersTree.Refresh() // counts changed in place; Set would reset selection and branches
		a.ma.callOnUpdate()
	})
}

func (a *mailsNavigationPane) MakeFolderMenu() []*fyne.MenuItem {
	var items1 []*fyne.MenuItem
	a.foldersTree.Data().Walk(nil, func(f *mailFolderNode) bool {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			go a.ma.MessagePane.setCurrentFolder(context.Background(), f)
		})
		items1 = append(items1, it)
		return true
	})
	return items1
}

type mailFolderItemWidget struct {
	widget.BaseWidget

	icon        *widget.Icon
	nameLabel   *widget.Label
	unreadLabel *widget.Label
}

func newMailFolderItemWidget() *mailFolderItemWidget {
	w := &mailFolderItemWidget{
		nameLabel:   widget.NewLabel(""),
		icon:        widget.NewIcon(icons.BlankSvg),
		unreadLabel: widget.NewLabel("999"),
	}
	w.ExtendBaseWidget(w)
	w.nameLabel.Truncation = fyne.TextTruncateClip
	return w
}

func (w *mailFolderItemWidget) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(nil, nil, w.icon, w.unreadLabel, w.nameLabel)
	return widget.NewSimpleRenderer(c)
}

func (w *mailFolderItemWidget) set(n *mailFolderNode) {
	w.icon.SetResource(n.icon())
	if n.UnreadCount == 0 {
		w.nameLabel.TextStyle.Bold = false
		w.unreadLabel.Hide()
	} else {
		w.nameLabel.TextStyle.Bold = true
		w.unreadLabel.SetText(strconv.Itoa(n.UnreadCount))
		w.unreadLabel.Show()
	}
	w.nameLabel.Text = n.Name
	w.nameLabel.Refresh()

}

type mailRow struct {
	characterID   int64
	characterName string
	from          *app.EveEntity
	id            int64
	isRead        bool // snapshot used for filtering, so rows don't vanish from an active Unread filter
	isRead2       bool // current state, used for display
	mailID        int64
	searchTarget  string
	subject       string
	timestamp     time.Time
}

// Filter options
const (
	mailsFilterCharacter    = "Character"
	mailsFilterFrom         = "From"
	mailsFilterStatus       = "Status"
	mailsFilterStatusRead   = "Read"
	mailsFilterStatusUnread = "Unread"
)

type mailsMessagePane struct {
	widget.BaseWidget

	OnSelected func()

	columnSorter  *xwidget.ColumnSorter[mailRow]
	currentFolder atomic.Pointer[mailFolderNode]
	errorLabel    *widget.Label
	filterChip    *xwidget.FilterChipCompact
	filterRun     latestRun
	footerLabel   *widget.Label
	headerList    *widget.List
	ma            *Mails
	reselecting   bool // suppresses OnSelected while restoring the selection
	rows          []mailRow
	rowsFolderUID widget.TreeNodeID // folder the rows were loaded for
	rowsFiltered  []mailRow
	searchEntry   *xwidget.SearchEntry
	sortChip      *kxwidget.SortChip
	topLabel      *widget.Label
	updateRun     latestRun
}

func newMailsMessagePane(ma *Mails) *mailsMessagePane {
	columnSorter := xwidget.NewColumnSorter(xwidget.NewDataColumns([]xwidget.DataColumn[mailRow]{{
		Label: "Date",
		Sort: func(a, b mailRow) int {
			return a.timestamp.Compare(b.timestamp)
		},
	}, {
		Label: "From",
		Sort: func(a, b mailRow) int {
			return strings.Compare(a.from.NameOrZero(), b.from.NameOrZero())
		},
	}, {
		Label: "Subject",
		Sort: func(a, b mailRow) int {
			return strings.Compare(a.subject, b.subject)
		},
	}}),
		"Date",
		xwidget.SortDesc,
	)
	a := &mailsMessagePane{
		columnSorter: columnSorter,
		errorLabel:   ui.NewLabelWithWrapping(""),
		footerLabel:  ui.NewLabelWithTruncation(""),
		ma:           ma,
		topLabel:     widget.NewLabel("?"),
	}
	a.ExtendBaseWidget(a)
	a.errorLabel.Importance = widget.DangerImportance
	a.errorLabel.Hide()
	a.headerList = a.makeHeaderList()
	a.topLabel.SizeName = theme.SizeNameSubHeadingText
	a.topLabel.Truncation = fyne.TextTruncateEllipsis
	a.searchEntry = xwidget.NewSearchEntry("Search mails", func(_ string) {
		a.filterRowsAsync()
	})
	a.sortChip = a.columnSorter.NewSortChip(func() {
		a.filterRowsAsync()
	})
	a.filterChip = xwidget.NewFilterChipCompact(nil, func(state map[string]string) {
		if f := a.currentFolder.Load(); f == nil || a.rowsFolderUID != f.UID() {
			return // rows are stale; update filters once the new folder is loaded
		}
		if state[mailsFilterStatus] != "" {
			a.updateIsRead()
		}
		a.filterRowsAsync()
	})
	a.setControlsEnabled(false) // until mails are loaded
	return a
}

func (a *mailsMessagePane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(
			a.topLabel,
			container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(a.filterChip, a.sortChip),
				a.searchEntry,
			),
		),
		container.New(layout.NewCustomPaddedVBoxLayout(0), a.errorLabel, a.footerLabel),
		nil,
		nil,
		a.headerList,
	)
	return widget.NewSimpleRenderer(c)
}

func (a *mailsMessagePane) makeHeaderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.rowsFiltered)
		},
		func() fyne.CanvasObject {
			return NewMailHeaderItemWidget(a.ma.u.EVEImage().EveEntityLogoAsync)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.rowsFiltered) {
				return
			}
			r := a.rowsFiltered[id]
			if a.ma.forCharacter && a.ma.character.Load() == nil {
				return
			}
			item := co.(*MailHeaderItemWidget)
			item.Set(r.from, r.subject, r.timestamp, r.isRead2)
		})
	l.OnSelected = func(id widget.ListItemID) {
		if a.reselecting || id >= len(a.rowsFiltered) {
			return
		}
		r := a.rowsFiltered[id]
		a.ma.ReadingPane.showMail(r)
		if !r.isRead2 && !a.ma.u.IsOffline() && !a.ma.u.IsUpdateDisabled() {
			a.markRead(r)
		}
		if a.OnSelected != nil {
			a.OnSelected()
			l.UnselectAll()
		}
	}
	return l
}

func (a *mailsMessagePane) clear() {
	a.filterRun.start() // discards pending filter results
	xslices.Clear(&a.rows)
	xslices.Clear(&a.rowsFiltered)
	a.rowsFolderUID = ""
	a.headerList.Refresh()
	a.topLabel.SetText("?")
	a.errorLabel.Hide()
	a.setFooter("", widget.MediumImportance)
}

func (a *mailsMessagePane) setFooter(s string, i widget.Importance) {
	a.footerLabel.Text = s
	a.footerLabel.Importance = i
	a.footerLabel.Refresh()
}

// setInfo shows an info instead of mails, e.g. when they are not loaded yet.
func (a *mailsMessagePane) setInfo(s string, i widget.Importance) {
	a.setFooter(s, i)
	a.setControlsEnabled(false)
}

func (a *mailsMessagePane) setError(err error) {
	a.errorLabel.SetText("ERROR: " + a.ma.u.ErrorDisplay(err))
	a.errorLabel.Show()
	a.setControlsEnabled(false)
}

func (a *mailsMessagePane) setControlsEnabled(enabled bool) {
	if enabled {
		a.searchEntry.Enable()
		a.filterChip.Enable()
		a.sortChip.Enable()
		return
	}
	a.searchEntry.Disable()
	a.filterChip.Disable()
	a.sortChip.Disable()
}

func (a *mailsMessagePane) setCurrentFolder(ctx context.Context, folder *mailFolderNode) {
	a.currentFolder.Store(folder)
	fyne.Do(func() {
		a.headerList.ScrollToOffset(0) // ScrollToTop panics on an unrendered list
		a.headerList.UnselectAll()
		a.ma.ReadingPane.clear()
		a.searchEntry.ClearSilent()
		a.filterChip.Reset()
	})
	a.update(ctx)
}

// markRead marks a mail as read. It is updated locally first, so the UI reflects it right away.
func (a *mailsMessagePane) markRead(r mailRow) {
	runAsync(func() {
		a.ma.sig.Do(fmt.Sprintf("charactermails-set-read-%d-%d", r.characterID, r.mailID), func() (any, error) {
			ctx := context.Background()
			err := a.ma.u.Character().SetMailReadLocal(ctx, r.characterID, r.mailID, true)
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", r.characterID, "mailID", r.mailID, "error", err)
				return nil, nil
			}
			a.ma.mailsChanged(ctx, r.characterID)
			err = a.ma.u.Character().UpdateMailRead(ctx, r.characterID, r.mailID, true)
			if err != nil {
				slog.Error("Failed to mark mail as read on ESI", "characterID", r.characterID, "mailID", r.mailID, "error", err)
				a.ma.u.DisplaySnackbar("ERROR: Failed to mark mail as read: " + r.subject)
				a.ma.mailsChanged(ctx, r.characterID) // local state was reset
			}
			return nil, nil
		})
	})
}

func (a *mailsMessagePane) updateIsRead() {
	for i := range a.rows {
		a.rows[i].isRead = a.rows[i].isRead2
	}
}

// update refreshes the headers for the current folder.
func (a *mailsMessagePane) update(ctx context.Context) {
	isLatest := a.updateRun.start()
	folder := a.currentFolder.Load()
	reset := func() {
		a.clear()
		a.ma.ReadingPane.clear()
	}
	if folder == nil {
		return // cleared by Mails.update, which also shows the status
	}

	rows, err := a.fetchRows(ctx, folder)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to refresh mail headers UI", "characterID", folder.CharacterID, "folder", folder.Name, "err", err)
		fyne.Do(func() {
			if !isLatest() || !a.isCurrentFolder(folder) {
				return
			}
			reset() // first, because it hides the error
			a.setError(err)
		})
		return
	}
	fyne.Do(func() {
		a.applyRows(isLatest, folder, rows)
	})
}

// applyRows shows the headers of a folder unless a newer update was started
// or the folder is no longer shown. Must be called on the main thread.
func (a *mailsMessagePane) applyRows(isLatest func() bool, folder *mailFolderNode, rows []mailRow) {
	if !isLatest() || !a.isCurrentFolder(folder) {
		return
	}
	if a.rowsFolderUID == folder.UID() {
		snapshot := make(map[int64]bool, len(a.rows))
		for _, r := range a.rows {
			snapshot[r.id] = r.isRead
		}
		for i, r := range rows {
			if v, ok := snapshot[r.id]; ok {
				rows[i].isRead = v
			}
		}
	}
	a.errorLabel.Hide()
	a.setControlsEnabled(true)
	a.topLabel.SetText(folder.Name)
	a.rows = rows
	a.rowsFolderUID = folder.UID()
	a.filterRowsAsync()
}

// isCurrentFolder reports whether folder is still the one to show, e.g. not of a previous character.
func (a *mailsMessagePane) isCurrentFolder(folder *mailFolderNode) bool {
	current := a.currentFolder.Load()
	if current == nil || current.UID() != folder.UID() {
		return false
	}
	return !a.ma.forCharacter || folder.CharacterID == a.ma.character.Load().IDOrZero()
}

func (a *mailsMessagePane) filterRowsAsync() {
	isLatest := a.filterRun.start()
	rows := slices.Clone(a.rows)
	totalRows := len(rows)
	filter := a.filterChip.Selected()
	search := strings.ToLower(a.searchEntry.Text)
	sortCol, dir, doSort := a.columnSorter.CalcSort("")
	runAsync(func() {
		// filter
		if x := filter[mailsFilterStatus]; x != "" {
			switch x {
			case mailsFilterStatusUnread:
				rows = slices.DeleteFunc(rows, func(r mailRow) bool {
					return r.isRead
				})
			case mailsFilterStatusRead:
				rows = slices.DeleteFunc(rows, func(r mailRow) bool {
					return !r.isRead
				})
			}
		}
		if x := filter[mailsFilterCharacter]; x != "" {
			rows = slices.DeleteFunc(rows, func(r mailRow) bool {
				return r.characterName != x
			})
		}
		if x := filter[mailsFilterFrom]; x != "" {
			rows = slices.DeleteFunc(rows, func(r mailRow) bool {
				return r.from.NameOrZero() != x
			})
		}
		if len(search) > 1 {
			rows = slices.DeleteFunc(rows, func(r mailRow) bool {
				return !strings.Contains(r.searchTarget, search)
			})
		}

		// sort
		a.columnSorter.SortRows(rows, sortCol, dir, doSort)

		// collect options
		characterOptions := xslices.Map(rows, func(r mailRow) string {
			return r.characterName
		})
		fromOptions := xslices.Map(rows, func(r mailRow) string {
			return r.from.NameOrZero()
		})
		statusOptions := xslices.Map(rows, func(r mailRow) string {
			if r.isRead2 {
				return mailsFilterStatusRead
			}
			return mailsFilterStatusUnread
		})

		footer := fmt.Sprintf(
			"Showing %s / %s messages",
			ihumanize.Comma(len(rows)),
			ihumanize.Comma(totalRows),
		)
		options := []xwidget.FilterOption{
			xwidget.NewFilterOptionMultiChoice(mailsFilterStatus, statusOptions),
		}
		if !a.ma.forCharacter {
			options = append(options, xwidget.NewFilterOptionMultiChoice(mailsFilterCharacter, characterOptions))
		}
		options = append(options, xwidget.NewFilterOptionMultiChoice(mailsFilterFrom, fromOptions))

		fyne.Do(func() {
			if !isLatest() {
				return
			}
			a.setFooter(footer, widget.MediumImportance)
			a.filterChip.SetOptions(options...)
			a.rowsFiltered = rows
			a.headerList.Refresh()
			a.syncSelection()
		})
	})
}

// syncSelection keeps the displayed mail selected after the rows changed
// and clears it when it is no longer shown.
func (a *mailsMessagePane) syncSelection() {
	requested := a.ma.ReadingPane.requested
	if requested.mailID == 0 {
		return
	}
	idx := slices.IndexFunc(a.rowsFiltered, func(r mailRow) bool {
		return r.characterID == requested.characterID && r.mailID == requested.mailID
	})
	if idx == -1 {
		a.headerList.UnselectAll()
		a.ma.ReadingPane.clear()
		return
	}
	if a.OnSelected != nil {
		return // mobile does not keep a selection
	}
	a.reselecting = true
	a.headerList.Select(idx)
	a.reselecting = false
}

func (a *mailsMessagePane) fetchRows(ctx context.Context, f *mailFolderNode) ([]mailRow, error) {
	cs := a.ma.u.Character()
	var hh []*app.CharacterMailHeader
	var err error
	switch {
	case a.ma.forCharacter && f.Category == nodeCategoryLabel:
		hh, err = cs.ListMailHeadersForLabelOrdered(ctx, f.CharacterID, f.ObjID)
	case a.ma.forCharacter && f.Category == nodeCategoryList:
		hh, err = cs.ListMailHeadersForListOrdered(ctx, f.CharacterID, f.ObjID)
	case f.Category == nodeCategoryLabel:
		hh, err = cs.ListAllMailHeadersForLabelOrdered(ctx, f.ObjID)
	case f.Category == nodeCategoryList:
		hh, err = cs.ListAllMailHeadersForListOrdered(ctx, f.ObjID)
	}
	if err != nil {
		return nil, err
	}
	var characterNames map[int64]string
	if !a.ma.forCharacter {
		characterNames, err = cs.CharacterNames(ctx)
		if err != nil {
			return nil, err
		}
	}
	rows := make([]mailRow, len(hh))
	for i, h := range hh {
		characterName := characterNames[h.CharacterID]
		rows[i] = mailRow{
			characterID:   h.CharacterID,
			characterName: characterName,
			from:          h.From,
			id:            h.ID,
			isRead:        h.IsRead,
			isRead2:       h.IsRead,
			mailID:        h.MailID,
			searchTarget:  strings.ToLower(h.Subject + "-" + h.From.NameOrZero() + "-" + characterName),
			subject:       h.Subject,
			timestamp:     h.Timestamp,
		}
	}
	return rows, nil
}

type mailsReadingPane struct {
	widget.BaseWidget

	// OnCleared is called on the main thread when a shown or requested mail is cleared.
	OnCleared func()

	bodyLabel    *widget.Label
	headerWidget *MailHeaderWidget
	ma           *Mails
	mail         *app.CharacterMail
	owner        *app.EveEntity
	requested    struct{ characterID, mailID int64 } // latest mail requested for display
	subjectLabel *widget.Label
	toolbar      *widget.Toolbar
}

func newMailsReadingPane(ma *Mails) *mailsReadingPane {
	a := &mailsReadingPane{
		bodyLabel:    widget.NewLabel(""),
		headerWidget: NewMailHeaderWidget(ma.u.EVEImage().EveEntityLogoAsync, ma.u.InfoViewer().Show),
		ma:           ma,
		subjectLabel: widget.NewLabel(""),
	}
	a.ExtendBaseWidget(a)
	a.subjectLabel.SizeName = theme.SizeNameSubHeadingText
	a.subjectLabel.Truncation = fyne.TextTruncateClip
	a.subjectLabel.Selectable = true
	a.bodyLabel.Wrapping = fyne.TextWrapWord
	a.bodyLabel.Selectable = true
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	return a
}

func (a *mailsReadingPane) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(a.subjectLabel, a.headerWidget),
		nil,
		nil,
		nil,
		ui.NewWideScrollBarOverride(container.NewVScroll(a.bodyLabel)),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *mailsReadingPane) MakeDeleteAction() (fyne.Resource, func()) {
	return theme.DeleteIcon(), func() {
		// callback runs off the main thread, so it must not read a.mail
		m := a.mail
		if m == nil {
			return
		}
		subject := xstrings.TruncateWithSuffix(m.Subject.ValueOrFallback("?"), 50, 0)
		ui.ShowLoadingConfirm(
			a.ma.u,
			"Delete mail?",
			fmt.Sprintf(
				"You are about to permanently delete \"%s\" from %s. "+
					"This action cannot be undone and the mail cannot be recovered.",
				subject,
				m.From.Name,
			),
			"Delete",
			widget.DangerImportance,
			func(ctx context.Context) {
				err := a.ma.u.Character().DeleteMail(ctx, m.CharacterID, m.MailID)
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					slog.Error("Failed to delete mail",
						slog.Int64("characterID", m.CharacterID),
						slog.Int64("mailID", m.MailID),
						slog.Any("err", err),
					)
					a.ma.u.DisplaySnackbar(fmt.Sprintf("Failed to delete mail \"%s\": %s", subject, a.ma.u.ErrorDisplay(err)))
					return
				}
				a.ma.mailsChanged(ctx, m.CharacterID) // clears the reading pane, which calls OnCleared
				a.ma.u.DisplaySnackbar(fmt.Sprintf("Mail \"%s\" deleted", subject))
			}, a.ma.u.MainWindow(),
		)
	}
}

func (a *mailsReadingPane) MakeForwardAction() (fyne.Resource, func()) {
	return theme.MailForwardIcon(), func() {
		if a.mail == nil {
			return
		}
		a.ma.showMailerWindow(mailer.Forward, a.mail)
	}
}

func (a *mailsReadingPane) MakeReplyAction() (fyne.Resource, func()) {
	return theme.MailReplyIcon(), func() {
		if a.mail == nil {
			return
		}
		a.ma.showMailerWindow(mailer.Reply, a.mail)
	}
}

func (a *mailsReadingPane) MakeReplyAllAction() (fyne.Resource, func()) {
	return theme.MailReplyAllIcon(), func() {
		if a.mail == nil {
			return
		}
		a.ma.showMailerWindow(mailer.ReplyAll, a.mail)
	}
}

func (a *mailsReadingPane) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(a.MakeReplyAction()),
		widget.NewToolbarAction(a.MakeReplyAllAction()),
		widget.NewToolbarAction(a.MakeForwardAction()),
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if a.mail == nil {
				return
			}
			fyne.CurrentApp().Clipboard().SetContent(a.mail.String())
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(a.MakeDeleteAction()),
	)
	return toolbar
}

// showMail displays the mail of a row and discards results from earlier requests.
func (a *mailsReadingPane) showMail(r mailRow) {
	a.reset() // a replaced mail is not a cleared one
	if r.characterID == 0 {
		return
	}
	if !a.ma.forCharacter {
		a.owner = &app.EveEntity{
			Category: app.EveEntityCharacter,
			ID:       r.characterID,
			Name:     r.characterName,
		}
	}
	a.requested.characterID, a.requested.mailID = r.characterID, r.mailID
	runAsync(func() {
		a.loadMail(context.Background(), r.characterID, r.mailID)
	})
}

func (a *mailsReadingPane) isRequested(characterID, mailID int64) bool {
	return a.requested.characterID == characterID && a.requested.mailID == mailID
}

// clear removes the mail and notifies OnCleared if one was shown or requested.
func (a *mailsReadingPane) clear() {
	hadMail := a.requested.mailID != 0
	a.reset()
	if hadMail && a.OnCleared != nil {
		a.OnCleared()
	}
}

func (a *mailsReadingPane) reset() {
	a.mail = nil
	a.requested.characterID, a.requested.mailID = 0, 0
	a.subjectLabel.SetText("")
	a.headerWidget.Clear()
	a.bodyLabel.SetText("")
	a.owner = nil
	a.toolbar.Hide()
}

func (a *mailsReadingPane) setMail(m *app.CharacterMail) {
	a.subjectLabel.SetText(m.Subject.ValueOrZero())
	a.setBody(m.BodyPlain())
	a.headerWidget.Set(m.From, m.Timestamp, a.owner, m.Recipients...)
}

func (a *mailsReadingPane) setBody(s string) {
	var i widget.Importance
	if s == "" {
		i = widget.LowImportance
		s = "Loading..."
	}
	a.bodyLabel.Importance = i
	a.bodyLabel.Text = s
	a.bodyLabel.Refresh()
}

func (a *mailsReadingPane) loadMail(ctx context.Context, characterID, mailID int64) {
	mail, err := a.ma.u.Character().GetMail(ctx, characterID, mailID)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		fyne.Do(func() {
			if !a.isRequested(characterID, mailID) {
				return
			}
			a.setBody("ERROR: Failed to load: " + a.ma.u.ErrorDisplay(err))
		})
		return
	}
	fyne.Do(func() {
		if !a.isRequested(characterID, mailID) {
			return
		}
		a.mail = mail
		a.setMail(mail)
		a.toolbar.Show()
	})

	if a.ma.u.IsOffline() || a.ma.u.IsUpdateDisabled() {
		return
	}

	// try to fetch mail body if missing
	if mail.Body.IsEmpty() {
		go func() {
			a.ma.sig.Do(fmt.Sprintf("charactermails-load-mail-%d-%d", characterID, mailID), func() (any, error) {
				body, err := a.ma.u.Character().UpdateMailBodyESI(ctx, characterID, mail.MailID)
				if err != nil {
					if ctx.Err() != nil {
						return nil, nil
					}
					slog.Error("Failed to update mail body", "characterID", characterID, "mailID", mail.MailID, "error", err)
					fyne.Do(func() {
						if a.mail == nil || a.mail.CharacterID != characterID || a.mail.MailID != mailID {
							return
						}
						a.setBody("ERROR: Failed to load: " + a.ma.u.ErrorDisplay(err))
					})
					return nil, nil
				}
				fyne.Do(func() {
					if a.mail == nil || a.mail.CharacterID != characterID || a.mail.MailID != mailID {
						return
					}
					a.mail.Body.Set(body)
					a.setBody(a.mail.BodyPlain())
				})
				return nil, nil
			})
		}()
	}
}
