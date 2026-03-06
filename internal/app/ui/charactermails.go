package ui

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	kxmodal "github.com/ErikKalkoken/fyne-kx/modal"
	kwidget "github.com/ErikKalkoken/fyne-kx/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/ErikKalkoken/evebuddy/internal/app"
	"github.com/ErikKalkoken/evebuddy/internal/app/icons"
	"github.com/ErikKalkoken/evebuddy/internal/app/xdialog"
	"github.com/ErikKalkoken/evebuddy/internal/xslices"
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
	folderNodeUnread
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

func (f mailFolderNode) IsEmpty() bool {
	return f.CharacterID == 0
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

type characterMails struct {
	widget.BaseWidget

	Detail  *mailDetail
	Headers *fyne.Container

	character        atomic.Pointer[app.Character]
	compose          *widget.Button
	currentFolder    atomic.Pointer[mailFolderNode]
	folderDefault    *mailFolderNode
	folderDownloaded *ttwidget.Label
	folders          *xwidget.Tree[mailFolderNode]
	folderStatus     *widget.Label
	folderTotal      *widget.Label
	headerList       *widget.List
	headers          []*app.CharacterMailHeader
	headerStatus     *widget.Label
	headerTop        *widget.Label
	lastFolder       *mailFolderNode
	lastSelected     widget.ListItemID
	mail             *app.CharacterMail
	missingPercent   atomic.Int64
	onSelected       func()
	onSendMessage    func(character *app.Character, mode app.SendMailMode, mail *app.CharacterMail)
	onUpdate         func(unread, missing int)
	toolbar          *widget.Toolbar
	u                *baseUI
	unreadCount      atomic.Int64
}

func newCharacterMails(u *baseUI) *characterMails {
	a := &characterMails{
		Detail:           newMailDetail(u),
		folderDownloaded: ttwidget.NewLabel(""),
		folderStatus:     widget.NewLabel(""),
		folderTotal:      widget.NewLabel("?"),
		headerStatus:     widget.NewLabel(""),
		headerTop:        widget.NewLabel(""),
		u:                u,
	}
	a.ExtendBaseWidget(a)

	// Folders
	a.folders = a.makeFolderTree()
	a.folderStatus.Hide()
	r, f := a.makeComposeMessageAction()
	a.compose = widget.NewButtonWithIcon("Compose", r, f)
	a.compose.Importance = widget.HighImportance
	a.compose.Disable()

	// Headers
	a.headerStatus.Hide()
	a.headerList = a.makeHeaderList()
	a.Headers = container.NewBorder(
		container.NewVBox(
			a.headerTop,
			a.headerStatus,
		),
		nil,
		nil,
		nil,
		a.headerList,
	)

	// Detail
	a.toolbar = a.makeToolbar()
	a.toolbar.Hide()

	a.u.signals.CurrentCharacterExchanged.AddListener(func(ctx context.Context, c *app.Character) {
		a.character.Store(c)
		a.update(ctx)
	})
	a.u.signals.CharacterSectionChanged.AddListener(func(ctx context.Context, arg app.CharacterSectionUpdated) {
		if characterIDOrZero(a.character.Load()) != arg.CharacterID {
			return
		}
		switch arg.Section {
		case
			app.SectionCharacterMailLabels,
			app.SectionCharacterMailLists,
			app.SectionCharacterMailHeaders:
			a.update(ctx)
		}
	})
	a.u.signals.RefreshTickerExpired.AddListener(func(ctx context.Context, _ struct{}) {
		a.updateDownloaded(ctx)
	})
	return a
}

func (a *characterMails) CreateRenderer() fyne.WidgetRenderer {
	split1 := container.NewHSplit(
		a.Headers,
		container.NewBorder(a.toolbar, nil, nil, nil, a.Detail),
	)
	split1.SetOffset(0.35)

	folders := container.NewBorder(
		container.NewVBox(container.NewPadded(a.compose), a.folderStatus),
		container.NewHBox(a.folderTotal, layout.NewSpacer(), a.folderDownloaded),
		nil,
		nil,
		a.folders,
	)
	split2 := container.NewHSplit(folders, split1)
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

func (a *characterMails) makeFolderTree() *xwidget.Tree[mailFolderNode] {
	t := xwidget.NewTree(
		func(isBranch bool) fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(icons.BlankSvg),
				widget.NewLabel("template template"),
				layout.NewSpacer(),
				kwidget.NewBadge("999"),
			)
		},
		func(n *mailFolderNode, b bool, co fyne.CanvasObject) {
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
	t.OnSelectedNode = func(n *mailFolderNode) {
		if n.isBranch() {
			t.UnselectAll()
			t.ToggleBranchNode(n)
			return
		}
		a.lastFolder = n
		go a.setCurrentFolder(context.Background(), n)
	}
	return t
}

func (a *characterMails) update(ctx context.Context) {
	clearAll := func() {
		fyne.Do(func() {
			a.folders.Clear()
			a.currentFolder.Store(nil)
			a.headers = xslices.Reset(a.headers)
			a.headerList.Refresh()
			a.headerTop.SetText("")
			a.clearMail()
			a.folderTotal.SetText("?")
			a.folderDownloaded.SetText("")
		})
	}
	setStatus := func(s string, i widget.Importance) {
		fyne.Do(func() {
			a.folderStatus.Text = s
			a.folderStatus.Importance = i
			a.folderStatus.Refresh()
			a.folderStatus.Show()
			a.compose.Disable()
		})
	}
	characterID := characterIDOrZero(a.character.Load())
	if characterID == 0 {
		clearAll()
		setStatus("No character", widget.LowImportance)
		return
	}
	hasData := a.u.scs.HasCharacterSection(characterID, app.SectionCharacterMailHeaders)
	if !hasData {
		clearAll()
		setStatus("Data not fully loaded yet", widget.WarningImportance)
		return
	}
	td, folderAll, err := a.fetchFolders(ctx, characterID)
	if err != nil {
		slog.Error("Failed to build mail tree", "character", characterID, "error", err)
		setStatus("Error: "+app.ErrorDisplay(err), widget.DangerImportance)
		return
	}
	unread, err := a.updateCountsInTree(ctx, characterID, td)
	if err != nil {
		slog.Error("Failed to update mail counts", "character", characterID, "error", err)
	} else {
		folderAll.UnreadCount = unread
	}
	a.setCurrentFolder(ctx, folderAll)
	fyne.Do(func() {
		a.compose.Enable()
		a.folderStatus.Hide()
		a.folders.Set(td)
		a.folders.SelectNode(folderAll)
	})
	a.unreadCount.Store(int64(folderAll.UnreadCount))
	a.updateDownloaded(ctx)
	fyne.Do(func() {
		a.callOnUpdate()
	})
}

func (a *characterMails) callOnUpdate() {
	if a.onUpdate == nil {
		return
	}
	a.onUpdate(int(a.unreadCount.Load()), int(a.missingPercent.Load()))
}

func (a *characterMails) updateDownloaded(ctx context.Context) {
	var total2, downloaded, hint string
	var missingPercent int
	func() {
		characterID := characterIDOrZero(a.character.Load())
		if characterID == 0 {
			return
		}
		total, missing, err := a.u.cs.DownloadedBodiesPercentage(ctx, characterID)
		if err != nil {
			slog.Error("updateDownloaded", "error", err)
			total2 = "ERROR"
			return
		}
		p := message.NewPrinter(language.English)
		total2 = p.Sprintf("%d total", total)
		if total == 0 || missing == 0 {
			return
		}

		missingPercent = int(float64(missing) / float64(total) * 100)
		downloaded = fmt.Sprintf("%d%% downloaded", 100-missingPercent)
		hint = p.Sprintf("%d missing", missing)
	}()
	a.missingPercent.Store(int64(missingPercent))
	fyne.Do(func() {
		a.folderTotal.SetText(total2)
		if downloaded == "" {
			a.folderDownloaded.Hide()
			return
		}
		a.folderDownloaded.SetText(downloaded)
		a.folderDownloaded.SetToolTip(hint)
		a.folderDownloaded.Show()
		a.callOnUpdate()
	})
}

func (a *characterMails) fetchFolders(ctx context.Context, characterID int64) (xwidget.TreeData[mailFolderNode], *mailFolderNode, error) {
	var td xwidget.TreeData[mailFolderNode]
	if characterID == 0 {
		return td, nil, nil
	}

	// Add unread folder
	err := td.Add(nil, &mailFolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeUnread,
		Name:        "Unread",
		ObjID:       app.MailLabelUnread,
	}, false)
	if err != nil {
		return td, nil, err
	}

	// Add default folders
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
		err := td.Add(nil, &mailFolderNode{
			CharacterID: characterID,
			Category:    nodeCategoryLabel,
			Type:        o.nodeType,
			Name:        o.name,
			ObjID:       o.labelID,
		}, false)
		if err != nil {
			return td, nil, err
		}
	}

	// Add custom labels
	labels, err := a.u.cs.ListMailLabelsOrdered(ctx, characterID)
	if err != nil {
		return td, nil, err
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
	lists, err := a.u.cs.ListMailLists(ctx, characterID)
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
	folderAll := &mailFolderNode{
		Category:    nodeCategoryLabel,
		CharacterID: characterID,
		Type:        folderNodeAll,
		Name:        "All",
		ObjID:       app.MailLabelAll,
	}
	err = td.Add(nil, folderAll, false)
	if err != nil {
		return td, nil, err
	}
	return td, folderAll, nil
}

func (a *characterMails) updateUnreadCounts(ctx context.Context) {
	td := a.folders.Data()
	characterID := characterIDOrZero(a.character.Load())
	unread, err := a.updateCountsInTree(ctx, characterID, td)
	if err != nil {
		slog.Error("Failed to update unread counts", "characterID", characterID, "error", err)
		return
	}
	a.unreadCount.Store(int64(unread))
	fyne.Do(func() {
		a.folders.Set(td)
		a.callOnUpdate()
	})
}

func (a *characterMails) updateCountsInTree(ctx context.Context, characterID int64, td xwidget.TreeData[mailFolderNode]) (int, error) {
	if td.IsEmpty() {
		return 0, nil
	}
	labelUnreadCounts, err := a.u.cs.GetMailLabelUnreadCounts(ctx, characterID)
	if err != nil {
		return 0, err
	}
	listUnreadCounts, err := a.u.cs.GetMailListUnreadCounts(ctx, characterID)
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

	td.Walk(nil, func(n *mailFolderNode) bool {
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
		}
		return true
	})
	return totalCount, nil
}

func (a *characterMails) makeFolderMenu() []*fyne.MenuItem {
	// current := u.MailArea.CurrentFolder.ValueOrZero()
	var items1 []*fyne.MenuItem
	a.folders.Data().Walk(nil, func(f *mailFolderNode) bool {
		s := f.Name
		if f.UnreadCount > 0 {
			s += fmt.Sprintf(" (%d)", f.UnreadCount)
		}
		it := fyne.NewMenuItem(s, func() {
			go a.setCurrentFolder(context.Background(), f)
		})
		// if f == current {
		// 	it.Disabled = true
		// }
		items1 = append(items1, it)
		return true
	})
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
			if a.character.Load() == nil {
				return
			}
			item := co.(*mailHeaderItem)
			item.Set(characterIDOrZero(a.character.Load()), m.From, m.Subject, m.Timestamp, m.IsRead)
		})
	l.OnSelected = func(id widget.ListItemID) {
		if id >= len(a.headers) {
			return
		}
		r := a.headers[id]
		go a.loadMail(context.Background(), r.MailID)
		a.lastSelected = id
		if a.onSelected != nil {
			a.onSelected()
			l.UnselectAll()
		}
	}
	return l
}

func (a *characterMails) resetCurrentFolder(ctx context.Context) {
	a.setCurrentFolder(ctx, a.folderDefault)
}

func (a *characterMails) setCurrentFolder(ctx context.Context, folder *mailFolderNode) {
	a.currentFolder.Store(folder)

	a.headerUpdate(ctx)
	fyne.Do(func() {
		a.headerList.ScrollToTop()
		a.headerList.UnselectAll()
		a.clearMail()
	})
}

func (a *characterMails) headerUpdate(ctx context.Context) {
	clear := func() {
		fyne.Do(func() {
			a.headers = xslices.Reset(a.headers)
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
	folder := a.currentFolder.Load()
	if folder == nil {
		clear()
		return
	}
	hasData := a.u.scs.HasCharacterSection(folder.CharacterID, app.SectionCharacterMailHeaders)
	if !hasData {
		setStatus("Data not yet loaded", widget.WarningImportance)
		clear()
		return
	}

	headers, err := a.fetchHeaders(ctx, folder)
	if err != nil {
		slog.Error("Failed to refresh mail headers UI", "characterID", folder.CharacterID, "folder", folder.Name, "err", err)
		setStatus("Failed to load: "+app.ErrorDisplay(err), widget.DangerImportance)
		clear()
		return
	}

	p := message.NewPrinter(language.English)
	s := p.Sprintf("%s • %d mails", folder.Name, len(headers))
	fyne.Do(func() {
		a.headerStatus.Hide()
		a.headerTop.SetText(s)
		a.headers = headers
		a.headerList.Refresh()
	})
}

func (a *characterMails) fetchHeaders(ctx context.Context, f *mailFolderNode) ([]*app.CharacterMailHeader, error) {
	var h []*app.CharacterMailHeader
	var err error
	switch f.Category {
	case nodeCategoryLabel:
		h, err = a.u.cs.ListMailHeadersForLabelOrdered(ctx, f.CharacterID, f.ObjID)
	case nodeCategoryList:
		h, err = a.u.cs.ListMailHeadersForListOrdered(ctx, f.CharacterID, f.ObjID)
	}
	return h, err
}

func (a *characterMails) doOnSendMessage(mode app.SendMailMode, mail *app.CharacterMail) {
	if a.onSendMessage == nil {
		return
	}
	c := a.character.Load()
	if c == nil {
		return
	}
	a.onSendMessage(c, mode, mail)
}

func (a *characterMails) makeComposeMessageAction() (fyne.Resource, func()) {
	return theme.DocumentCreateIcon(), func() {
		a.doOnSendMessage(app.SendMailNew, nil)
	}
}

func (a *characterMails) MakeDeleteAction(onSuccess func()) (fyne.Resource, func()) {
	return theme.DeleteIcon(), func() {
		xdialog.ShowConfirm(
			"Delete mail",
			fmt.Sprintf("Are you sure you want to permanently delete this mail?\n\n%s", a.mail.Header()),
			"Delete",
			func(confirmed bool) {
				if !confirmed {
					return
				}
				ctx := context.Background()
				m := kxmodal.NewProgressInfinite(
					"Deleting mail...",
					"",
					func() error {
						return a.u.cs.DeleteMail(ctx, a.mail.CharacterID, a.mail.MailID)
					},
					a.u.MainWindow(),
				)
				m.OnSuccess = func() {
					a.headerUpdate(ctx)
					if onSuccess != nil {
						onSuccess()
					}
					a.u.ShowSnackbar(fmt.Sprintf("Mail \"%s\" deleted", a.mail.Subject))
				}
				m.OnError = func(err error) {
					slog.Error("Failed to delete mail", "characterID", a.mail.CharacterID, "mailID", a.mail.MailID, "err", err)
					a.u.ShowSnackbar(fmt.Sprintf("Failed to delete mail: %s", app.ErrorDisplay(err)))
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
			fyne.CurrentApp().Clipboard().SetContent(a.mail.String())
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

func (a *characterMails) loadMail(ctx context.Context, mailID int64) {
	characterID := characterIDOrZero(a.character.Load())
	if characterID == 0 {
		return
	}
	mail, err := a.u.cs.GetMail(ctx, characterID, mailID)
	if err != nil {
		slog.Error("Failed to fetch mail", "mailID", mailID, "error", err)
		fyne.Do(func() {
			a.Detail.SetBody("ERROR: Failed to load: " + app.ErrorDisplay(err))
		})
		return
	}
	fyne.Do(func() {
		a.mail = mail
		a.Detail.SetMail(mail)
		a.toolbar.Show()
	})

	if app.IsOfflineMode() || a.u.isUpdateDisabled.Load() {
		return
	}

	// try to fetch mail body if missing
	if mail.Body.IsEmpty() {
		go func() {
			a.u.sig.Do(fmt.Sprintf("charactermails-load-mail-%d-%d", characterID, mailID), func() (any, error) {
				body, err := a.u.cs.UpdateMailBodyESI(ctx, characterID, mail.MailID)
				if err != nil {
					slog.Error("Failed to update mail body", "characterID", characterID, "mailID", mail.MailID, "error", err)
					fyne.Do(func() {
						if a.mail.CharacterID != characterID || a.mail.MailID != mailID {
							return
						}
						a.Detail.SetBody("ERROR: Failed to load: " + app.ErrorDisplay(err))
					})
					return nil, nil
				}
				fyne.Do(func() {
					if a.mail.CharacterID != characterID || a.mail.MailID != mailID {
						return
					}
					a.mail.Body.Set(body)
					a.Detail.SetBody(a.mail.BodyPlain())
				})
				return nil, nil
			})
		}()
	}

	// try to update mail as read if unread
	if !mail.IsRead.ValueOrZero() {
		go func() {
			a.u.sig.Do(fmt.Sprintf("charactermails-set-read-%d-%d", characterID, mailID), func() (any, error) {
				err := a.u.cs.UpdateMailRead(ctx, characterID, mail.MailID, true)
				if err != nil {
					slog.Error("Failed to mark mail as read", "characterID", characterID, "mailID", mail.MailID, "error", err)
					a.u.ShowSnackbar("ERROR: Failed to mark mail as read: " + mail.Subject.ValueOrZero())
					return nil, nil
				}
				a.updateUnreadCounts(ctx)
				a.headerUpdate(ctx)
				a.u.characterOverview.updateItem(ctx, characterID)
				a.u.updateMailIndicator(ctx)
				fyne.Do(func() {
					if a.mail.CharacterID != characterID || a.mail.MailID != mailID {
						return
					}
					a.mail.IsRead.Set(true)
				})
				return nil, nil
			})
		}()
	}
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
		header:  newMailHeader(u.eis, u.InfoWindow().ShowEntity),
		subject: widget.NewLabel(""),
	}
	w.subject.SizeName = theme.SizeNameSubHeadingText
	w.subject.Truncation = fyne.TextTruncateClip
	w.subject.Selectable = true
	w.body.Wrapping = fyne.TextWrapWord
	w.body.Selectable = true
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
	w.subject.SetText(m.Subject.ValueOrZero())
	w.SetBody(m.BodyPlain())
	w.header.Set(m.CharacterID, m.From, m.Timestamp, m.Recipients...)
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
