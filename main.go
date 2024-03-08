package main

import (
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"example/esiapp/internal/core"
	"example/esiapp/internal/storage"

	"github.com/microcosm-cc/bluemonday"
)

const (
	myDateTime = "2006-01-02 15:04"
)

var characterID int32

type labelItem struct {
	id   int32
	name string
}

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if err := storage.Initialize(); err != nil {
		log.Fatal(err)
	}

	// storage.Test()

	myApp := app.New()
	myWindow := myApp.NewWindow("Eve Online App")

	c, err := storage.FetchFirstCharacter()
	if err != nil {
		log.Printf("Failed to load any character: %v", err)
	} else {
		characterID = c.ID
	}

	user := makeUserSegment(myWindow)
	mails := makeMailsSegment()

	// folder segment
	refreshButton := makeRefreshButton(myWindow)
	folders, data := makeFolders()

	labels, err := storage.FetchAllMailLabels(characterID)
	if err != nil {
		log.Fatal(err)
	}
	for _, l := range labels {
		data.Append(labelItem{id: l.ID, name: l.Name})
	}

	folderSegment := container.NewBorder(refreshButton, nil, nil, nil, folders)

	main := container.NewHSplit(folderSegment, mails)
	main.SetOffset(0.15)

	content := container.NewBorder(user, nil, nil, nil, main)
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()
}

func makeUserSegment(myWindow fyne.Window) *fyne.Container {
	shareItem := makeShareItem()
	buttonAdd := newContextMenuButton(
		"Manage Characters", fyne.NewMenu("",
			fyne.NewMenuItem("Add Character", func() {
				info := dialog.NewInformation(
					"Add Character",
					"Please follow instructions in your browser to add a new character.",
					myWindow,
				)
				info.Show()
				_, err := core.AddCharacter()
				if err != nil {
					log.Printf("Failed to add a new character: %v", err)
				}
			}),
			shareItem,
		))

	currentUser := container.NewHBox()
	c, err := storage.FetchCharacter(characterID)
	if err != nil {
		currentUser.Add(widget.NewLabel("No characters"))
		log.Print("No token found")
	} else {
		image := canvas.NewImageFromURI(c.PortraitURL(64))
		image.FillMode = canvas.ImageFillOriginal
		currentUser.Add(image)
		currentUser.Add(widget.NewLabel(c.Name))
		characterID = c.ID
	}
	currentUser.Add(layout.NewSpacer())
	currentUser.Add(buttonAdd)
	return currentUser
}

func makeShareItem() *fyne.MenuItem {
	cc, err := storage.FetchAllCharacters()
	if err != nil {
		log.Fatal(err)
	}
	shareItem := fyne.NewMenuItem("Switch character", nil)

	var items []*fyne.MenuItem
	for _, c := range cc {
		item := fyne.NewMenuItem(c.Name, func() { log.Printf("selected %v", c.Name) })
		items = append(items, item)
	}
	shareItem.ChildMenu = fyne.NewMenu("", items...)
	return shareItem
}

func makeRefreshButton(w fyne.Window) *widget.Button {
	b := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if characterID == 0 {
			info := dialog.NewInformation(
				"Warning",
				"Please select a character first.",
				w,
			)
			info.Show()
			return
		}
		err := core.UpdateMails(characterID)
		if err != nil {
			log.Fatal(err)
		}
	})
	return b
}

func makeFolders() (fyne.CanvasObject, binding.ExternalUntypedList) {
	var ii []interface{}
	ii = append(ii, labelItem{id: 0, name: "All Mails"})
	data := binding.BindUntypedList(&ii)
	folders := widget.NewListWithData(
		data,
		func() fyne.CanvasObject {
			return widget.NewLabel("from")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			if entry, err := i.(binding.Untyped).Get(); err != nil {
				log.Println("Failed to Get item")
			} else {
				w := o.(*widget.Label)
				w.SetText(entry.(labelItem).name)
			}
		})

	folders.OnSelected = func(id widget.ListItemID) {
		d, _ := data.Get()
		n := d[id].(labelItem).name
		log.Printf("Selected label %v", n)
	}

	return folders, data
}

func makeMailsSegment() fyne.CanvasObject {
	mails, err := storage.FetchMailsForLabel(characterID, 0)
	if err != nil {
		log.Fatalf("Failed to fetch mail: %v", err)
	}
	mailHeaderSubject := widget.NewLabel("")
	mailHeaderSubject.TextStyle = fyne.TextStyle{Bold: true}
	mailHeaderSubject.Truncation = fyne.TextTruncateEllipsis

	mailHeaderBlock := widget.NewLabel("")
	mailHeader := container.NewVBox(mailHeaderSubject, mailHeaderBlock)
	mailBody := widget.NewLabel("")
	mailBody.Wrapping = fyne.TextWrapBreak

	detail := container.NewBorder(mailHeader, nil, nil, nil, container.NewVScroll(mailBody))

	headersTotal := widget.NewLabel(fmt.Sprintf("%d mails", len(mails)))
	blue := bluemonday.StrictPolicy()
	headersList := widget.NewList(
		func() int {
			return len(mails)
		},
		func() fyne.CanvasObject {
			from := widget.NewLabel("from")
			timestamp := widget.NewLabel("timestamp")
			subject := widget.NewLabel("subject")
			subject.TextStyle = fyne.TextStyle{Bold: true}
			subject.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(
				container.NewHBox(from, layout.NewSpacer(), timestamp),
				subject,
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			mail := mails[i]
			// style := fyne.TextStyle{Bold: !mail.IsRead}
			parent := o.(*fyne.Container)
			top := parent.Objects[0].(*fyne.Container)
			from := top.Objects[0].(*widget.Label)
			from.SetText(mail.From.Name)
			timestamp := top.Objects[2].(*widget.Label)
			timestamp.SetText(mail.TimeStamp.Format(myDateTime))
			subject := parent.Objects[1].(*widget.Label)
			subject.SetText(mail.Subject)
		})
	headersList.OnSelected = func(id widget.ListItemID) {
		mail := mails[id]
		mailHeaderSubject.SetText(mail.Subject)
		var names []string
		for _, n := range mail.Recipients {
			names = append(names, n.Name)
		}
		mailHeaderBlock.SetText("From: " + mail.From.Name + "\nSent: " + mail.TimeStamp.Format(myDateTime) + "\nTo: " + strings.Join(names, ", "))
		text := strings.ReplaceAll(mail.Body, "<br>", "\n")
		mailBody.SetText(blue.Sanitize(text))
	}

	headers := container.NewBorder(headersTotal, nil, nil, nil, headersList)

	mainMails := container.NewHSplit(headers, detail)
	mainMails.SetOffset(0.35)

	return mainMails
}
