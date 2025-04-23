package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	appwidget "github.com/ErikKalkoken/evebuddy/internal/app/widget"
	"github.com/ErikKalkoken/evebuddy/internal/optional"
	iwidget "github.com/ErikKalkoken/evebuddy/internal/widget"
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
	folderNodeUnread
)

// A FolderNode in the folder tree, e.g. the inbox
type FolderNode struct {
	Category    folderNodeCategory
	CharacterID int32
	IsLeaf      bool
	Type        folderNodeType
	Name        string
	ObjID       int32
	UnreadCount int
}

func (f FolderNode) IsEmpty() bool {
	return f.CharacterID == 0
}

func (f FolderNode) UID() widget.TreeNodeID {
	if f.CharacterID == 0 || f.Type == folderNodeUndefined {
		panic("some IDs are not set")
	}
	return fmt.Sprintf("%d-%d-%d", f.CharacterID, f.Type, f.ObjID)
}

func (f FolderNode) isBranch() bool {
	return f.Category == nodeCategoryBranch
}

func (f FolderNode) icon() fyne.Resource {
	switch f.Type {
	case folderNodeInbox:
		return theme.DownloadIcon()
	case folderNodeSent:
		return theme.UploadIcon()
	case folderNodeTrash:
		return theme.DeleteIcon()
	}
	return theme.FolderIcon()
}

var emptyFolder = FolderNode{}

type CharacterMails struct {
	widget.BaseWidget

	CurrentFolder optional.Optional[FolderNode]
	Detail        fyne.CanvasObject
	Headers       fyne.CanvasObject
	OnSelected    func()
	OnUpdate      func(count int)
	OnSendMessage func(character *app.Character, mode app.SendMailMode, mail *app.CharacterMail)

	body          *widget.Label
	folders       *iwidget.Tree[FolderNode]
	header        *MailHeader
	headerList    *widget.List
	headers       []*app.CharacterMailHeader
	headerTop     *widget.Label
	folderDefault FolderNode
	lastSelected  widget.ListItemID
	lastFolder    FolderNode
	mail          *app.CharacterMail
	subject       *iwidget.Label
	toolbar       *widget.Toolbar
	u             *BaseUI
}

func NewCharacterMails(u *BaseUI) *CharacterMails {
	a := &CharacterMails{
		body:      widget.NewLabel(""),
		header:    NewMailHeader(u.EveImageService(), u.ShowEveEntityInfoWindow),
		headers:   make([]*app.CharacterMailHeader, 0),
		headerTop: appwidget.MakeTopLabel(),
		subject:   iwidget.NewLabelWithSize("", theme.SizeNameSubHeadingText),
		u:         u,
	}
	a.ExtendBaseWidget(a)

	// Mail
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()
	a.subject.Truncation = fyne.TextTruncateClip
	a.body.Wrapping = fyne.TextWrapWord
	a.Detail = container.NewBorder(container.NewVBox(a.subject, a.header), nil, nil, nil, container.NewVScroll(a.body))

	// Headers
	a.headerList = a.makeHeaderList()
	a.Headers = container.NewBorder(a.headerTop, nil, nil, nil, a.headerList)

	// Folders
	a.folders = a.makeFolderTree()
	return a
}

func (a *CharacterMails) CreateRenderer() fyne.WidgetRenderer {
	detailWithToolbar := container.NewBorder(a.toolbar, nil, nil, nil, a.Detail)
	split1 := container.NewHSplit(a.Headers, detailWithToolbar)
	split1.SetOffset(0.35)

	r, f := a.MakeComposeMessageAction()
	compose := widget.NewButtonWithIcon("Compose", r, f)
	compose.Importance = widget.HighImportance

	split2 := container.NewHSplit(container.NewBorder(
		container.NewCenter(container.NewPadded(compose)),
		nil,
		nil,
		nil,
		a.folders,
	),
		split1,
	)
	split2.SetOffset(0.15)
	p := theme.Padding()
	c := container.NewBorder(
		widget.NewSeparator(),
		nil,
		nil,
		nil,
		container.New(layout.NewCustomPaddedLayout(-p, 0, 0, 0), split2),
	)
	return widget.NewSimpleRenderer(c)
}

func (a *CharacterMails) makeFolderTree() *iwidget.Tree[FolderNode] {
	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(icons.BlankSvg),
				widget.NewLabel("template template"),
				layout.NewSpacer(),
				kwidget.NewBadge("999"),
			)
		},
		func(n FolderNode, b bool, co fyne.CanvasObject) {
			hbox := co.(*fyne.Container).Objects
			icon := hbox[0].(*widget.Icon)
			icon.SetResource(n.icon())
			label := hbox[1].(*widget.Label)
			badge := hbox[3].(*kwidget.Badge)
			if n.UnreadCount == 0 {
				label.TextStyle.Bold = false
				badge.Hide()
			} else {
				label.TextStyle.Bold = true
				badge.SetText(strconv.Itoa(n.UnreadCount))
				badge.Show()
			}
			label.Text = n.Name
			label.Refresh()
		},
	)
	tree.OnSelected = func(n FolderNode) {
		if n.isBranch() {
			tree.UnselectAll()
			return
		}
		a.lastFolder = n
		a.setCurrentFolder(n)
	}
	return tree
}

func (a *CharacterMails) update() {
	a.lastFolder = emptyFolder
	a.update2()
}

func (a *CharacterMails) MakeFolderMenu() []*fyne.MenuItem {
	// current := u.MailArea.CurrentFolder.ValueOrZero()
	items1 := make([]*fyne.MenuItem, 0)
	for _, f := range a.folders.Data().Flat() {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			a.setCurrentFolder(f)
		})
		// if f == current {
		// 	it.Disabled = true
		// }
		items1 = append(items1, it)
	}
	return items1
}

func (a *CharacterMails) update2() {
	characterID := a.u.CurrentCharacterID()
	tree, folderAll, err := a.updateFolderData(characterID)
	if err != nil {
		t := "Failed to build folder tree"
		slog.Error(t, "character", characterID, "error", err)
		fyne.Do(func() {
			a.u.ShowErrorDialog(t, err, a.u.MainWindow())
		})
		return
	}
	if !folderAll.IsEmpty() {
		a.folderDefault = folderAll
	}
	fyne.Do(func() {
		a.folders.Set(tree)
	})
	fyne.Do(func() {
		if folderAll.IsEmpty() {
			a.clearFolder()
			return
		}
		if a.lastFolder != emptyFolder {
			a.headerRefresh()
			return
		}
		a.folders.UnselectAll()
		a.folders.ScrollToTop()
		a.folders.Select(folderAll)
		a.setCurrentFolder(folderAll)
	})
	if a.OnUpdate != nil {
		a.OnUpdate(folderAll.UnreadCount)
	}
}

func (a *CharacterMails) updateFolderData(characterID int32) (*iwidget.TreeData[FolderNode], FolderNode, error) {
	tree := iwidget.NewTreeData[FolderNode]()
	if characterID == 0 {
		return tree, emptyFolder, nil
	}
	ctx := context.Background()
	labelUnreadCounts, err := a.u.CharacterService().GetMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		return nil, FolderNode{}, err
	}
	listUnreadCounts, err := a.u.CharacterService().GetMailListUnreadCounts(ctx, characterID)
	if err != nil {
		return nil, FolderNode{}, err
	}
	totalUnreadCount, totalLabelsUnreadCount, totalListUnreadCount := calcUnreadTotals(labelUnreadCounts, listUnreadCounts)

	// Add unread folder
	folderUnread := FolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeUnread,
		Name:        "Unread",
		ObjID:       app.MailLabelUnread,
		UnreadCount: totalUnreadCount,
	}
	tree.MustAdd(iwidget.RootUID, folderUnread)

	// Add default folders
	defaultFolders := []struct {
		nodeType folderNodeType
		labelID  int32
		name     string
	}{
		{folderNodeInbox, app.MailLabelInbox, "Inbox"},
		{folderNodeSent, app.MailLabelSent, "Sent"},
		{folderNodeCorp, app.MailLabelCorp, "Corp"},
		{folderNodeAlliance, app.MailLabelAlliance, "Alliance"},
	}
	for _, o := range defaultFolders {
		u, ok := labelUnreadCounts[o.labelID]
		if !ok {
			u = 0
		}
		n := FolderNode{
			CharacterID: characterID,
			Category:    nodeCategoryLabel,
			Type:        o.nodeType,
			Name:        o.name,
			ObjID:       o.labelID,
			UnreadCount: u,
		}
		tree.MustAdd(iwidget.RootUID, n)
	}

	// Add custom labels
	labels, err := a.u.CharacterService().ListMailLabelsOrdered(ctx, characterID)
	if err != nil {
		return nil, FolderNode{}, err
	}
	if len(labels) > 0 {
		n := FolderNode{
			CharacterID: characterID,
			Type:        folderNodeLabel,
			Name:        "Labels",
			UnreadCount: totalLabelsUnreadCount,
		}
		uid := tree.MustAdd(iwidget.RootUID, n)
		for _, l := range labels {
			u, ok := labelUnreadCounts[l.LabelID]
			if !ok {
				u = 0
			}
			n := FolderNode{
				Category:    nodeCategoryLabel,
				CharacterID: characterID,
				Name:        l.Name,
				ObjID:       l.LabelID,
				UnreadCount: u,
				Type:        folderNodeLabel,
			}
			tree.MustAdd(uid, n)
		}
	}

	// Add mailing lists
	lists, err := a.u.CharacterService().ListMailLists(ctx, characterID)
	if err != nil {
		return nil, FolderNode{}, err
	}
	if len(lists) > 0 {
		n := FolderNode{
			CharacterID: characterID,
			Type:        folderNodeList,
			Name:        "Mailing Lists",
			UnreadCount: totalListUnreadCount,
		}
		uid := tree.MustAdd(iwidget.RootUID, n)
		for _, l := range lists {
			u, ok := listUnreadCounts[l.ID]
			if !ok {
				u = 0
			}
			n := FolderNode{
				Category:    nodeCategoryList,
				CharacterID: characterID,
				ObjID:       l.ID,
				Name:        l.Name,
				UnreadCount: u,
				Type:        folderNodeList,
			}
			tree.MustAdd(uid, n)
		}
	}
	// Add all folder
	folderAll := FolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeAll,
		Name:        "All",
		ObjID:       app.MailLabelAll,
		UnreadCount: totalUnreadCount,
	}
	tree.MustAdd(iwidget.RootUID, folderAll)
	return tree, folderAll, nil
}

func calcUnreadTotals(labelCounts, listCounts map[int32]int) (int, int, int) {
	var total, labels, lists int
	for id, c := range labelCounts {
		total += c
		if id > app.MailLabelAlliance {
			labels += c
		}
	}
	for _, c := range listCounts {
		total += c
		lists += c
	}
	return total, labels, lists
}

func (a *CharacterMails) makeHeaderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.headers)
		},
		func() fyne.CanvasObject {
			return NewMailHeaderItem(a.u.EveImageService())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.headers) {
				return
			}
			m := a.headers[id]
			if !a.u.hasCharacter() {
				return
			}
			item := co.(*MailHeaderItem)
			item.Set(m.From, m.Subject, m.Timestamp, m.IsRead)
		})
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.headers) {
			return
		}
		r := a.headers[id]
		a.setMail(r.MailID)
		a.lastSelected = id
		if a.OnSelected != nil {
			a.OnSelected()
			l.UnselectAll()
		}
	}
	return l
}

func (a *CharacterMails) resetCurrentFolder() {
	a.setCurrentFolder(a.folderDefault)
}

func (a *CharacterMails) setCurrentFolder(folder FolderNode) {
	a.CurrentFolder = optional.New(folder)
	a.headerRefresh()
	a.headerList.ScrollToTop()
	a.headerList.UnselectAll()
	a.clearMail()
}

func (a *CharacterMails) clearFolder() {
	a.CurrentFolder = optional.Optional[FolderNode]{}
	a.headers = make([]*app.CharacterMailHeader, 0)
	a.headerList.Refresh()
	a.headerTop.SetText("")
	a.clearMail()
}

func (a *CharacterMails) headerRefresh() {
	var t string
	var i widget.Importance
	f, err := a.updateHeaders()
	if err != nil {
		slog.Error("Failed to refresh mail headers UI", "err", err)
		t = "ERROR"
		i = widget.DangerImportance
	} else {
		t, i = a.makeFolderTopText(f)
	}
	a.headerTop.Text = t
	a.headerTop.Importance = i
	a.headerTop.Refresh()
}

func (a *CharacterMails) updateHeaders() (FolderNode, error) {
	ctx := context.TODO()
	folderOption := a.CurrentFolder
	if folderOption.IsEmpty() {
		return FolderNode{}, nil
	}
	folder := folderOption.ValueOrZero()
	var headers []*app.CharacterMailHeader
	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		headers, err = a.u.CharacterService().ListMailHeadersForLabelOrdered(
			ctx,
			folder.CharacterID,
			folder.ObjID,
		)
	case nodeCategoryList:
		headers, err = a.u.CharacterService().ListMailHeadersForListOrdered(
			ctx,
			folder.CharacterID,
			folder.ObjID,
		)
	}
	if err != nil {
		return FolderNode{}, err
	}
	a.headers = headers
	a.headerList.Refresh()
	if len(headers) == 0 {
		a.clearMail()
	}
	return folder, nil
}

func (a *CharacterMails) makeFolderTopText(f FolderNode) (string, widget.Importance) {
	if !a.u.hasCharacter() {
		return "No Character", widget.LowImportance
	}
	hasData := a.u.StatusCacheService().CharacterSectionExists(a.u.CurrentCharacterID(), app.SectionSkillqueue)
	if !hasData {
		return "Waiting for character data to be loaded...", widget.WarningImportance
	}
	p := message.NewPrinter(language.English)
	s := p.Sprintf("%s • %d mails", f.Name, len(a.headers))
	return s, widget.MediumImportance
}

func (a *CharacterMails) onSendMessage(mode app.SendMailMode, mail *app.CharacterMail) {
	if a.OnSendMessage == nil {
		return
	}
	character := a.u.currentCharacter()
	if character == nil {
		return
	}
	a.OnSendMessage(character, mode, mail)
}

func (a *CharacterMails) MakeComposeMessageAction() (fyne.Resource, func()) {
	return theme.DocumentCreateIcon(), func() {
		a.onSendMessage(app.SendMailNew, nil)
	}
}

func (a *CharacterMails) MakeDeleteAction(onSuccess func()) (fyne.Resource, func()) {
	return theme.DeleteIcon(), func() {
		a.u.ShowConfirmDialog(
			"Delete mail",
			fmt.Sprintf("Are you sure you want to permanently delete this mail?\n\n%s", a.mail.Header()),
			"Delete",
			func(confirmed bool) {
				if !confirmed {
					return
				}
				m := kxmodal.NewProgressInfinite(
					"Deleting mail...",
					"",
					func() error {
						return a.u.CharacterService().DeleteMail(context.TODO(), a.mail.CharacterID, a.mail.MailID)
					},
					a.u.MainWindow(),
				)
				m.OnSuccess = func() {
					a.headerRefresh()
					if onSuccess != nil {
						onSuccess()
					}
					a.u.ShowSnackbar(fmt.Sprintf("Mail \"%s\" deleted", a.mail.Subject))
				}
				m.OnError = func(err error) {
					slog.Error("Failed to delete mail", "characterID", a.mail.CharacterID, "mailID", a.mail.MailID, "err", err)
					a.u.ShowSnackbar(fmt.Sprintf("Failed to delete mail: %s", a.u.ErrorDisplay(err)))
				}
				m.Start()
			}, a.u.MainWindow())
	}
}

func (a *CharacterMails) MakeForwardAction() (fyne.Resource, func()) {
	return theme.MailForwardIcon(), func() {
		a.onSendMessage(app.SendMailForward, a.mail)
	}
}

func (a *CharacterMails) MakeReplyAction() (fyne.Resource, func()) {
	return theme.MailReplyIcon(), func() {
		a.onSendMessage(app.SendMailReply, a.mail)
	}
}

func (a *CharacterMails) MakeReplyAllAction() (fyne.Resource, func()) {
	return theme.MailReplyAllIcon(), func() {
		a.onSendMessage(app.SendMailReplyAll, a.mail)
	}
}

func (a *CharacterMails) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(a.MakeReplyAction()),
		widget.NewToolbarAction(a.MakeReplyAllAction()),
		widget.NewToolbarAction(a.MakeForwardAction()),
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			a.u.App().Clipboard().SetContent(a.mail.String())
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(a.MakeDeleteAction(nil)),
	)
	return toolbar
}

func (a *CharacterMails) clearMail() {
	a.subject.SetText("")
	a.header.Clear()
	a.body.SetText("")
	a.toolbar.Hide()
}

func (a *CharacterMails) setMail(mailID int32) {
	ctx := context.TODO()
	characterID := a.u.CurrentCharacterID()
	var err error
	a.mail, err = a.u.CharacterService().GetMail(ctx, characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		a.u.ShowSnackbar("ERROR: Failed to fetch mail")
		return
	}
	if !a.u.IsOffline() && !a.mail.IsRead {
		go func() {
			err = a.u.CharacterService().UpdateMailRead(ctx, characterID, a.mail.MailID)
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
				a.u.ShowSnackbar("ERROR: Failed to mark mail as read")
				return
			}
			a.update2()
			a.u.updateCrossPages()
			a.u.updateMailIndicator()
		}()
	}
	a.subject.SetText(a.mail.Subject)
	a.header.Set(a.mail.From, a.mail.Timestamp, a.mail.Recipients...)
	a.body.SetText(a.mail.BodyPlain())
	a.toolbar.Show()
}
