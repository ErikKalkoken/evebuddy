package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

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

// A mailFolderNode in the folder tree, e.g. the inbox
type mailFolderNode struct {
	Category    folderNodeCategory
	CharacterID int32
	IsLeaf      bool
	Type        folderNodeType
	Name        string
	ObjID       int32
	UnreadCount int
}

func (f mailFolderNode) IsEmpty() bool {
	return f.CharacterID == 0
}

func (f mailFolderNode) UID() widget.TreeNodeID {
	return makeMailNodeUID(f.CharacterID, f.Type, f.ObjID)
}

func makeMailNodeUID(characterID int32, nodeType folderNodeType, objID int32) widget.TreeNodeID {
	if characterID == 0 || nodeType == folderNodeUndefined {
		panic("invalid IDs")
	}
	return fmt.Sprintf("%d-%d-%d", characterID, nodeType, objID)
}

func (f mailFolderNode) isBranch() bool {
	return f.Category == nodeCategoryBranch
}

func (f mailFolderNode) icon() fyne.Resource {
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

var emptyFolder = mailFolderNode{}

type characterMails struct {
	widget.BaseWidget

	Detail  *mailDetail
	Headers *fyne.Container

	character     *app.Character
	folderDefault mailFolderNode
	folders       *iwidget.Tree[mailFolderNode]
	folderTop     *widget.Label
	headerList    *widget.List
	headers       []*app.CharacterMailHeader
	headerStatus  *widget.Label
	headerTop     *widget.Label
	lastFolder    mailFolderNode
	lastSelected  widget.ListItemID
	mail          *app.CharacterMail
	onSelected    func()
	onSendMessage func(character *app.Character, mode app.SendMailMode, mail *app.CharacterMail)
	onUpdate      func(count int)
	toolbar       *widget.Toolbar
	u             *baseUI

	mu            sync.RWMutex
	currentFolder optional.Optional[mailFolderNode]
}

func newCharacterMails(u *baseUI) *characterMails {
	a := &characterMails{
		Detail:       newMailDetail(u),
		folderTop:    makeTopLabel(),
		headers:      make([]*app.CharacterMailHeader, 0),
		headerStatus: widget.NewLabel(""),
		headerTop:    makeTopLabel(),
		u:            u,
	}
	a.ExtendBaseWidget(a)

	// Folders
	a.folders = a.makeFolderTree()
	a.folderTop.Hide()

	// Headers
	a.headerStatus.Hide()
	a.headerList = a.makeHeaderList()
	a.Headers = container.NewBorder(
		a.headerTop,
		a.headerStatus,
		nil,
		nil,
		a.headerList,
	)

	// Detail
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()

	a.u.currentCharacterExchanged.AddListener(
		func(_ context.Context, c *app.Character) {
			a.character = c
		},
	)
	a.u.characterSectionChanged.AddListener(func(_ context.Context, arg characterSectionUpdated) {
		if characterIDOrZero(a.character) != arg.characterID {
			return
		}
		switch arg.section {
		case app.SectionCharacterMailLabels, app.SectionCharacterMailLists, app.SectionCharacterMailHeaders:
			a.update()
		}
	})
	return a
}

func (a *characterMails) CreateRenderer() fyne.WidgetRenderer {
	split1 := container.NewHSplit(
		a.Headers,
		container.NewBorder(a.toolbar, nil, nil, nil, a.Detail),
	)
	split1.SetOffset(0.35)

	r, f := a.makeComposeMessageAction()
	compose := widget.NewButtonWithIcon("Compose", r, f)
	compose.Importance = widget.HighImportance

	split2 := container.NewHSplit(container.NewBorder(
		container.NewVBox(
			layout.NewSpacer(),
			container.NewPadded(compose),
			a.folderTop,
			layout.NewSpacer(),
		),
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

func (a *characterMails) makeFolderTree() *iwidget.Tree[mailFolderNode] {
	tree := iwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(icons.BlankSvg),
				widget.NewLabel("template template"),
				layout.NewSpacer(),
				kwidget.NewBadge("999"),
			)
		},
		func(n mailFolderNode, b bool, co fyne.CanvasObject) {
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
	tree.OnSelectedNode = func(n mailFolderNode) {
		if n.isBranch() {
			tree.UnselectAll()
			tree.ToggleBranch(n.UID())
			return
		}
		a.lastFolder = n
		a.setCurrentFolder(n)
	}
	return tree
}

func (a *characterMails) update() {
	characterID := characterIDOrZero(a.character)
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterMailHeaders)
	var td iwidget.TreeData[mailFolderNode]
	folderAll := mailFolderNode{}
	var err error
	if hasData {
		td2, f, err2 := a.fetchFolders(a.u.services(), characterID)
		if err2 != nil {
			slog.Error("Failed to build mail tree", "character", characterID, "error", err2)
		} else {
			td = td2
			folderAll = f
			err = err2
		}
		total, err3 := a.updateCountsInTree(a.u.services(), characterID, td)
		if err3 != nil {
			slog.Error("Failed to update mail counts", "character", characterID, "error", err3)
		} else {
			folderAll.UnreadCount = total
		}
	}
	t, i := a.u.makeTopText(characterID, hasData, err, func() (string, widget.Importance) {
		return "", widget.MediumImportance
	})
	fyne.Do(func() {
		if t != "" {
			a.folderTop.Text, a.folderTop.Importance = t, i
			a.folderTop.Refresh()
			a.folderTop.Show()
		} else {
			a.folderTop.Hide()
		}
	})
	fyne.Do(func() {
		if !folderAll.IsEmpty() {
			a.folderDefault = folderAll
		}
		a.folders.Set(td)
	})
	fyne.Do(func() {
		if folderAll.IsEmpty() {
			a.clearFolder()
			return
		}
		a.folders.UnselectAll()
		a.folders.ScrollToTop()
		a.folders.Select(folderAll.UID())
		a.setCurrentFolder(folderAll)
	})
	if a.onUpdate != nil {
		a.onUpdate(folderAll.UnreadCount)
	}
}

func (*characterMails) fetchFolders(s services, characterID int32) (iwidget.TreeData[mailFolderNode], mailFolderNode, error) {
	var td iwidget.TreeData[mailFolderNode]
	if characterID == 0 {
		return td, emptyFolder, nil
	}

	// Add unread folder
	td.MustAdd(iwidget.TreeRootID, mailFolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeUnread,
		Name:        "Unread",
		ObjID:       app.MailLabelUnread,
	})

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
		td.MustAdd(iwidget.TreeRootID, mailFolderNode{
			CharacterID: characterID,
			Category:    nodeCategoryLabel,
			Type:        o.nodeType,
			Name:        o.name,
			ObjID:       o.labelID,
		})
	}

	// Add custom labels
	ctx := context.Background()
	labels, err := s.cs.ListMailLabelsOrdered(ctx, characterID)
	if err != nil {
		return td, mailFolderNode{}, err
	}
	if len(labels) > 0 {
		uid := td.MustAdd(iwidget.TreeRootID, mailFolderNode{
			Category:    nodeCategoryBranch,
			CharacterID: characterID,
			Name:        "Labels",
			Type:        folderNodeLabel,
		})
		for _, l := range labels {
			td.MustAdd(uid, mailFolderNode{
				Category:    nodeCategoryLabel,
				CharacterID: characterID,
				Name:        l.Name,
				ObjID:       l.LabelID,
				Type:        folderNodeLabel,
			})
		}
	}

	// Add mailing lists
	lists, err := s.cs.ListMailLists(ctx, characterID)
	if err != nil {
		return td, mailFolderNode{}, err
	}
	if len(lists) > 0 {
		uid := td.MustAdd(iwidget.TreeRootID, mailFolderNode{
			Category:    nodeCategoryBranch,
			CharacterID: characterID,
			Name:        "Mailing Lists",
			Type:        folderNodeList,
		})
		for _, l := range lists {
			td.MustAdd(uid, mailFolderNode{
				Category:    nodeCategoryList,
				CharacterID: characterID,
				ObjID:       l.ID,
				Name:        l.Name,
				Type:        folderNodeList,
			})
		}
	}
	// Add all folder
	folderAll := mailFolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeAll,
		Name:        "All",
		ObjID:       app.MailLabelAll,
	}
	td.MustAdd(iwidget.TreeRootID, folderAll)
	return td, folderAll, nil
}

func (a *characterMails) updateUnreadCounts() {
	td := a.folders.Data()
	characterID := characterIDOrZero(a.character)
	unread, err := a.updateCountsInTree(a.u.services(), characterID, td)
	if err != nil {
		slog.Error("Failed to update unread counts", "characterID", characterID, "error", err)
		return
	}
	fyne.Do(func() {
		a.folders.Set(td)
	})
	if a.onUpdate != nil {
		a.onUpdate(unread)
	}
}

func (*characterMails) updateCountsInTree(s services, characterID int32, td iwidget.TreeData[mailFolderNode]) (int, error) {
	if td.IsEmpty() {
		return 0, nil
	}
	ctx := context.Background()
	labelUnreadCounts, err := s.cs.GetMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		return 0, err
	}
	listUnreadCounts, err := s.cs.GetMailListUnreadCounts(ctx, characterID)
	if err != nil {
		return 0, err
	}

	var totalCount, labelCount, listCount int
	for id, c := range labelUnreadCounts {
		totalCount += c
		if id > app.MailLabelAlliance {
			labelCount += c
		}
	}
	for _, c := range listUnreadCounts {
		totalCount += c
		listCount += c
	}

	for n := range td.All() {
		var c int
		switch n.Type {
		case folderNodeAll, folderNodeUnread:
			c = totalCount
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
			if err := td.Replace(n); err != nil {
				slog.Error("setting unread counts", "node", n, "err", err)
			}
		}
	}
	return totalCount, nil
}

func (a *characterMails) makeFolderMenu() []*fyne.MenuItem {
	// current := u.MailArea.CurrentFolder.ValueOrZero()
	items1 := make([]*fyne.MenuItem, 0)
	for f := range a.folders.Data().All() {
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

func (a *characterMails) makeHeaderList() *widget.List {
	l := widget.NewList(
		func() int {
			return len(a.headers)
		},
		func() fyne.CanvasObject {
			return newMailHeaderItem(a.u.eis)
		},
		func(id widget.ListItemID, co fyne.CanvasObject) {
			if id >= len(a.headers) {
				return
			}
			m := a.headers[id]
			if a.character == nil {
				return
			}
			item := co.(*mailHeaderItem)
			item.Set(m.From, m.Subject, m.Timestamp, m.IsRead)
		})
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.headers) {
			return
		}
		r := a.headers[id]
		a.loadMail(r.MailID)
		a.lastSelected = id
		if a.onSelected != nil {
			a.onSelected()
			l.UnselectAll()
		}
	}
	return l
}

func (a *characterMails) resetCurrentFolder() {
	a.setCurrentFolder(a.folderDefault)
}

func (a *characterMails) setCurrentFolder(folder mailFolderNode) {
	a.mu.Lock()
	a.currentFolder = optional.New(folder)
	a.mu.Unlock()

	a.headerRefresh()
	fyne.Do(func() {
		a.headerList.ScrollToTop()
		a.headerList.UnselectAll()
		a.clearMail()
	})
}

func (a *characterMails) clearFolder() {
	a.mu.Lock()
	a.currentFolder = optional.Optional[mailFolderNode]{}
	a.mu.Unlock()

	a.headers = make([]*app.CharacterMailHeader, 0)
	a.headerList.Refresh()
	a.headerTop.SetText("")
	a.clearMail()
}

func (a *characterMails) headerRefresh() {
	clearHeaders := func() {
		fyne.Do(func() {
			a.headers = make([]*app.CharacterMailHeader, 0)
			a.headerList.Refresh()
			a.headerTop.SetText("")
			a.clearMail()
		})
	}
	setStatus := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.headerStatus.Text = s
			a.headerStatus.Importance = i
			a.headerStatus.Refresh()
			a.headerStatus.Show()
		})
	}
	fyne.Do(func() {
	})
	a.mu.RLock()
	currentFolder := a.currentFolder
	a.mu.RUnlock()
	if currentFolder.IsEmpty() {
		clearHeaders()
		return
	}

	characterID := characterIDOrZero(a.character)
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterMailHeaders)
	if !hasData {
		setStatus("Data not yet loaded", widget.WarningImportance)
		clearHeaders()
		return
	}

	headers, err := a.fetchHeaders(currentFolder.MustValue(), a.u.services())
	if err != nil {
		slog.Error("Failed to refresh mail headers UI", "characterID", characterID, "err", err)
		setStatus("Failed to load: "+a.u.humanizeError(err), widget.DangerImportance)
		clearHeaders()
		return
	}

	f := currentFolder.ValueOrZero()
	p := message.NewPrinter(language.English)
	s := p.Sprintf("%s • %d mails", f.Name, len(headers))
	fyne.Do(func() {
		a.headerStatus.Hide()
		a.headerTop.SetText(s)
		a.headers = headers
		a.headerList.Refresh()
		a.clearMail()
	})
}

func (*characterMails) fetchHeaders(folder mailFolderNode, s services) ([]*app.CharacterMailHeader, error) {
	// return nil, app.ErrNotFound
	ctx := context.Background()
	var headers []*app.CharacterMailHeader
	var err error
	switch folder.Category {
	case nodeCategoryLabel:
		headers, err = s.cs.ListMailHeadersForLabelOrdered(
			ctx,
			folder.CharacterID,
			folder.ObjID,
		)
	case nodeCategoryList:
		headers, err = s.cs.ListMailHeadersForListOrdered(
			ctx,
			folder.CharacterID,
			folder.ObjID,
		)
	}
	return headers, err
}

func (a *characterMails) doOnSendMessage(mode app.SendMailMode, mail *app.CharacterMail) {
	if a.onSendMessage == nil {
		return
	}
	if a.character == nil {
		return
	}
	a.onSendMessage(a.character, mode, mail)
}

func (a *characterMails) makeComposeMessageAction() (fyne.Resource, func()) {
	return theme.DocumentCreateIcon(), func() {
		a.doOnSendMessage(app.SendMailNew, nil)
	}
}

func (a *characterMails) MakeDeleteAction(onSuccess func()) (fyne.Resource, func()) {
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
						return a.u.cs.DeleteMail(context.TODO(), a.mail.CharacterID, a.mail.MailID)
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
					a.u.ShowSnackbar(fmt.Sprintf("Failed to delete mail: %s", a.u.humanizeError(err)))
				}
				m.Start()
			}, a.u.MainWindow(),
		)
	}
}

func (a *characterMails) MakeForwardAction() (fyne.Resource, func()) {
	return theme.MailForwardIcon(), func() {
		a.doOnSendMessage(app.SendMailForward, a.mail)
	}
}

func (a *characterMails) MakeReplyAction() (fyne.Resource, func()) {
	return theme.MailReplyIcon(), func() {
		a.doOnSendMessage(app.SendMailReply, a.mail)
	}
}

func (a *characterMails) MakeReplyAllAction() (fyne.Resource, func()) {
	return theme.MailReplyAllIcon(), func() {
		a.doOnSendMessage(app.SendMailReplyAll, a.mail)
	}
}

func (a *characterMails) makeToolbar() *widget.Toolbar {
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

func (a *characterMails) clearMail() {
	a.Detail.clear()
	a.toolbar.Hide()
}

func (a *characterMails) loadMail(mailID int32) {
	ctx := context.TODO()
	characterID := characterIDOrZero(a.character)
	var err error
	a.mail, err = a.u.cs.GetMail(ctx, characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		a.u.ShowSnackbar("ERROR: Failed to fetch mail")
		return
	}
	if !a.u.IsOffline() {
		if a.mail.Body == "" {
			go func() {
				a.u.sig.Do(fmt.Sprintf("charactermails-load-mail-%d-%d", characterID, mailID), func() (any, error) {
					body, err := a.u.cs.UpdateMailBodyESI(ctx, characterID, a.mail.MailID)
					if err != nil {
						slog.Error("Failed to update mail body", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
						fyne.Do(func() {
							a.Detail.SetBody("Failed to load: " + a.u.humanizeError(err))
						})
						return nil, nil
					}
					fyne.Do(func() {
						if a.mail.CharacterID != characterID || a.mail.MailID != mailID {
							return
						}
						a.mail.Body = body
						a.Detail.SetBody(a.mail.BodyPlain())
					})
					return nil, nil
				})
			}()
		}
		if !a.mail.IsRead {
			go func() {
				a.u.sig.Do(fmt.Sprintf("charactermails-set-read-%d-%d", characterID, mailID), func() (any, error) {
					err := a.u.cs.UpdateMailRead(ctx, characterID, a.mail.MailID, true)
					if err != nil {
						slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", a.mail.MailID, "error", err)
						a.u.ShowSnackbar("ERROR: Failed to mark mail as read: " + a.mail.Subject)
						return nil, nil
					}
					a.updateUnreadCounts()
					a.headerRefresh()
					a.u.characterOverview.update()
					a.u.updateMailIndicator()
					fyne.Do(func() {
						if a.mail.CharacterID != characterID || a.mail.MailID != mailID {
							return
						}
						a.mail.IsRead = true
					})
					return nil, nil
				})
			}()
		}
	}
	a.Detail.SetMail(a.mail)
	a.toolbar.Show()
}

type mailDetail struct {
	widget.BaseWidget

	body    *widget.Label
	header  *mailHeader
	subject *widget.Label
}

func newMailDetail(u *baseUI) *mailDetail {
	w := &mailDetail{
		body:    widget.NewLabel(""),
		header:  newMailHeader(u.eis, u.ShowEveEntityInfoWindow),
		subject: widget.NewLabel(""),
	}
	w.subject.SizeName = theme.SizeNameSubHeadingText
	w.subject.Truncation = fyne.TextTruncateClip
	w.body.Wrapping = fyne.TextWrapWord
	w.ExtendBaseWidget(w)
	return w
}

func (w *mailDetail) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewBorder(
		container.NewVBox(w.subject, w.header),
		nil,
		nil,
		nil,
		container.NewVScroll(w.body),
	)
	return widget.NewSimpleRenderer(c)
}

func (w *mailDetail) clear() {
	w.subject.SetText("")
	w.header.Clear()
	w.body.SetText("")
}

func (w *mailDetail) SetMail(m *app.CharacterMail) {
	w.SetSubject(m.Subject)
	w.SetBody(m.BodyPlain())
	w.SetHeader(m.From, m.Timestamp, m.Recipients...)
}

func (w *mailDetail) SetBody(s string) {
	var i widget.Importance
	if s == "" {
		i = widget.LowImportance
		s = "Loading..."
	}
	w.body.Importance = i
	w.body.Text = s
	w.body.Refresh()
}

func (w *mailDetail) SetError(s string) {
	w.body.Importance = widget.DangerImportance
	w.body.Text = s
	w.body.Refresh()
}

func (w *mailDetail) SetSubject(s string) {
	w.subject.SetText(s)
}

func (w *mailDetail) SetHeader(from *app.EveEntity, timestamp time.Time, recipients ...*app.EveEntity) {
	w.header.Set(from, timestamp, recipients...)
}
