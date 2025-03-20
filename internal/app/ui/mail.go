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
	"github.com/ErikKalkoken/evebuddy/internal/humanize"
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

type CharacterMail struct {
	widget.BaseWidget

	CurrentFolder optional.Optional[FolderNode]
	Detail        fyne.CanvasObject
	Headers       fyne.CanvasObject
	OnSelected    func()
	OnUpdate      func(count int)
	OnSendMessage func(character *app.Character, mode SendMailMode, mail *app.CharacterMail)

	body          *widget.Label
	folderSection fyne.CanvasObject
	folders       *iwidget.Tree[FolderNode]
	header        *appwidget.MailHeader
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

func NewCharacterMail(u *BaseUI) *CharacterMail {
	a := &CharacterMail{
		body:      widget.NewLabel(""),
		header:    appwidget.NewMailHeader(u.ShowEveEntityInfoWindow),
		headers:   make([]*app.CharacterMailHeader, 0),
		headerTop: widget.NewLabel(""),
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
	r, f := a.MakeComposeMessageAction()
	newButton := widget.NewButtonWithIcon("Compose", r, f)
	newButton.Importance = widget.HighImportance
	top := container.NewHBox(layout.NewSpacer(), container.NewPadded(newButton), layout.NewSpacer())
	a.folderSection = container.NewBorder(top, nil, nil, nil, a.folders)
	return a
}

func (a *CharacterMail) CreateRenderer() fyne.WidgetRenderer {
	detailWithToolbar := container.NewBorder(a.toolbar, nil, nil, nil, a.Detail)
	split1 := container.NewHSplit(a.Headers, detailWithToolbar)
	split1.SetOffset(0.35)
	split2 := container.NewHSplit(a.folderSection, split1)
	split2.SetOffset(0.15)
	return widget.NewSimpleRenderer(split2)
}

func (a *CharacterMail) makeFolderTree() *iwidget.Tree[FolderNode] {
	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(icons.BlankSvg),
				widget.NewLabel("template"),
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
		a.SetFolder(n)
	}
	return tree
}

func (a *CharacterMail) Update() {
	a.lastFolder = emptyFolder
	a.update()
}

func (a *CharacterMail) Folders() []FolderNode {
	return a.folders.Data().Flat()
}

func (a *CharacterMail) update() {
	characterID := a.u.CurrentCharacterID()
	folderAll, err := a.updateFolderData(characterID)
	if err != nil {
		t := "Failed to build folder tree"
		slog.Error(t, "character", characterID, "error", err)
		a.u.ShowErrorDialog(t, err, a.u.Window)
		return
	}
	if a.OnUpdate != nil {
		a.OnUpdate(folderAll.UnreadCount)
	}
	a.folders.Refresh()
	if folderAll.IsEmpty() {
		a.clearFolder()
		return
	}
	if a.lastFolder == emptyFolder {
		a.folders.UnselectAll()
		a.folders.ScrollToTop()
		a.folders.Select(folderAll)
		a.SetFolder(folderAll)
	} else {
		a.headerRefresh()
	}
	a.folderDefault = folderAll
}

func (a *CharacterMail) updateFolderData(characterID int32) (FolderNode, error) {
	tree := iwidget.NewTreeData[FolderNode]()
	if characterID == 0 {
		a.folders.Clear()
		return emptyFolder, nil
	}
	ctx := context.Background()
	labelUnreadCounts, err := a.u.CharacterService().GetCharacterMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		a.folders.Clear()
		return emptyFolder, err
	}
	listUnreadCounts, err := a.u.CharacterService().GetCharacterMailListUnreadCounts(ctx, characterID)
	if err != nil {
		a.folders.Clear()
		return emptyFolder, err
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
	labels, err := a.u.CharacterService().ListCharacterMailLabelsOrdered(ctx, characterID)
	if err != nil {
		return FolderNode{}, err
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
	lists, err := a.u.CharacterService().ListCharacterMailLists(ctx, characterID)
	if err != nil {
		return FolderNode{}, err
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

	a.folders.Set(tree)
	return folderAll, nil
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

func (a *CharacterMail) makeHeaderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.headers)
		},
		func() fyne.CanvasObject {
			return appwidget.NewMailHeaderItem(a.u.EveImageService())
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.headers) {
				return
			}
			m := a.headers[id]
			if !a.u.HasCharacter() {
				return
			}
			item := co.(*appwidget.MailHeaderItem)
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

func (a *CharacterMail) ResetFolders() {
	a.SetFolder(a.folderDefault)
}

func (a *CharacterMail) SetFolder(folder FolderNode) {
	a.CurrentFolder = optional.New(folder)
	a.headerRefresh()
	a.headerList.ScrollToTop()
	a.headerList.UnselectAll()
	a.clearMail()
}

func (a *CharacterMail) clearFolder() {
	a.CurrentFolder = optional.Optional[FolderNode]{}
	a.headers = make([]*app.CharacterMailHeader, 0)
	a.headerList.Refresh()
	a.headerTop.SetText("")
	a.clearMail()
}

func (a *CharacterMail) headerRefresh() {
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

func (a *CharacterMail) updateHeaders() (FolderNode, error) {
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
		headers, err = a.u.CharacterService().ListCharacterMailHeadersForLabelOrdered(
			ctx,
			folder.CharacterID,
			folder.ObjID,
		)
	case nodeCategoryList:
		headers, err = a.u.CharacterService().ListCharacterMailHeadersForListOrdered(
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

func (a *CharacterMail) makeFolderTopText(f FolderNode) (string, widget.Importance) {
	if !a.u.HasCharacter() {
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

func (a *CharacterMail) onSendMessage(mode SendMailMode, mail *app.CharacterMail) {
	if a.OnSendMessage == nil {
		return
	}
	character := a.u.CurrentCharacter()
	if character == nil {
		return
	}
	a.OnSendMessage(character, mode, mail)
}

func (a *CharacterMail) MakeComposeMessageAction() (fyne.Resource, func()) {
	return theme.DocumentCreateIcon(), func() {
		a.onSendMessage(SendMailNew, nil)
	}
}

func (a *CharacterMail) MakeDeleteAction(onSuccess func()) (fyne.Resource, func()) {
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
						return a.u.CharacterService().DeleteCharacterMail(context.TODO(), a.mail.CharacterID, a.mail.MailID)
					},
					a.u.Window,
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
					a.u.ShowSnackbar(fmt.Sprintf("Failed to delete mail: %s", humanize.Error(err)))
				}
				m.Start()
			}, a.u.Window)
	}
}

func (a *CharacterMail) MakeForwardAction() (fyne.Resource, func()) {
	return theme.MailForwardIcon(), func() {
		a.onSendMessage(SendMailForward, a.mail)
	}
}

func (a *CharacterMail) MakeReplyAction() (fyne.Resource, func()) {
	return theme.MailReplyIcon(), func() {
		a.onSendMessage(SendMailReply, a.mail)
	}
}

func (a *CharacterMail) MakeReplyAllAction() (fyne.Resource, func()) {
	return theme.MailReplyAllIcon(), func() {
		a.onSendMessage(SendMailReplyAll, a.mail)
	}
}

func (a *CharacterMail) makeToolbar() *widget.Toolbar {
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(a.MakeReplyAction()),
		widget.NewToolbarAction(a.MakeReplyAllAction()),
		widget.NewToolbarAction(a.MakeForwardAction()),
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			a.u.Window.Clipboard().SetContent(a.mail.String())
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(a.MakeDeleteAction(nil)),
	)
	return toolbar
}

func (a *CharacterMail) clearMail() {
	a.subject.SetText("")
	a.header.Clear()
	a.body.SetText("")
	a.toolbar.Hide()
}

func (a *CharacterMail) setMail(mailID int32) {
	ctx := context.TODO()
	characterID := a.u.CurrentCharacterID()
	var err error
	a.mail, err = a.u.CharacterService().GetCharacterMail(ctx, characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		a.u.ShowSnackbar("ERROR: Failed to fetch mail")
		return
	}
	if !a.u.isOffline && !a.mail.IsRead {
		go func() {
			err = a.u.CharacterService().UpdateMailRead(ctx, characterID, a.mail.MailID)
			if err != nil {
				slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
				a.u.ShowSnackbar("ERROR: Failed to mark mail as read")
				return
			}
			a.update()
			a.u.CharacterOverview.Update()
			a.u.UpdateMailIndicator()
		}()
	}
	a.subject.SetText(a.mail.Subject)
	a.header.Set(a.u.EveImageService(), a.mail.From, a.mail.Timestamp, a.mail.Recipients...)
	a.body.SetText(a.mail.BodyPlain())
	a.toolbar.Show()
}
