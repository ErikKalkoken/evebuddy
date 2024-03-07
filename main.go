package main

import (
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if err := storage.Initialize(); err != nil {
		log.Fatal(err)
	}

	myApp := app.New()
	myWindow := myApp.NewWindow("Eve Online App")

	currentUser, characterID := makeCurrentUserSegment(myWindow)
	mails := makeMailsSegment(characterID)
	folders := makeFoldersSegment(myWindow, characterID)

	main := container.NewHSplit(folders, mails)
	main.SetOffset(0.15)

	buttonTest := widget.NewButton("Test", func() {
		c := myWindow.Canvas()
		p := widget.NewPopUp(widget.NewLabel("Hi there"), c)
		s := c.Content().Size()
		x := s.Width/2 - p.Size().Width/2
		y := s.Height/2 - p.Size().Height/2
		p.ShowAtPosition(fyne.NewPos(x, y))
		p.Show()
	})

	c := container.NewBorder(currentUser, buttonTest, nil, nil, main)
	myWindow.SetContent(c)
	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.ShowAndRun()
}

func makeCurrentUserSegment(myWindow fyne.Window) (*fyne.Container, int32) {
	characters, err := storage.FetchAllCharacters()
	if err != nil {
		log.Fatal(err)
	}
	shareItem := fyne.NewMenuItem("Switch character", nil)

	var items []*fyne.MenuItem
	for _, c := range characters {
		item := fyne.NewMenuItem(c.Name, func() { log.Printf("selected %v", c.Name) })
		items = append(items, item)
	}
	shareItem.ChildMenu = fyne.NewMenu("", items...)
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
	var characterID int32
	character, err := storage.FetchFirstCharacter()
	if err != nil {
		currentUser.Add(widget.NewLabel("No characters"))
		log.Print("No token found")
	} else {
		image := canvas.NewImageFromURI(character.PortraitURL(64))
		image.FillMode = canvas.ImageFillOriginal
		currentUser.Add(image)
		currentUser.Add(widget.NewLabel(character.Name))
		characterID = character.ID
	}
	currentUser.Add(layout.NewSpacer())
	currentUser.Add(buttonAdd)
	return currentUser, characterID
}

func makeFoldersSegment(myWindow fyne.Window, characterID int32) fyne.CanvasObject {
	folderActions := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		if characterID == 0 {
			info := dialog.NewInformation(
				"Warning",
				"Please select a character first.",
				myWindow,
			)
			info.Show()
			return
		}
		err := core.UpdateMails(characterID)
		if err != nil {
			log.Fatal(err)
		}
	})

	labels := []string{"All Mails", "Inbox", "Sent", "[Corp]", "[Alliance]"}
	folders := widget.NewList(
		func() int {
			return len(labels)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("from")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(labels[i])
		})

	foldersPage := container.NewBorder(folderActions, nil, nil, nil, folders)
	return foldersPage
}

func makeMailsSegment(characterID int32) fyne.CanvasObject {
	mails, err := storage.FetchAllMails(characterID)
	if err != nil {
		log.Fatalf("Failed to fetch mail: %v", err)
	}
	mailHeaderSubject := widget.NewLabel("")
	mailHeaderSubject.TextStyle = fyne.TextStyle{Bold: true}
	mailHeaderSubject.Truncation = fyne.TextTruncateEllipsis

	mailHeaderFrom := widget.NewLabel("")
	mailHeaderTimestamp := widget.NewLabel("")
	mailHeader := container.NewVBox(mailHeaderSubject, mailHeaderFrom, mailHeaderTimestamp)
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
		mailHeaderFrom.SetText("From: " + mail.From.Name)
		mailHeaderTimestamp.SetText("Received: " + mail.TimeStamp.Format(myDateTime))
		text := strings.ReplaceAll(mail.Body, "<br>", "\n")
		mailBody.SetText(blue.Sanitize(text))
	}

	headers := container.NewBorder(headersTotal, nil, nil, nil, headersList)

	mainMails := container.NewHSplit(headers, detail)
	mainMails.SetOffset(0.35)

	return mainMails
}
