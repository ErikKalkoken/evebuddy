package ui

import (
	"example/esiapp/internal/esi"
	"example/esiapp/internal/set"
	"example/esiapp/internal/sso"
	"example/esiapp/internal/storage"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type labelItem struct {
	id   int32
	name string
}

const maxMails = 1000

type folders struct {
	esiApp        *esiApp
	content       fyne.CanvasObject
	boundList     binding.ExternalUntypedList
	boundCharID   binding.ExternalInt
	headers       *headers
	list          *widget.List
	refreshButton *widget.Button
}

func (f *folders) addRefreshButton() {
	b := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		f.updateMails()
	})
	f.refreshButton = b
}

func (f *folders) updateMailsWithID(charID int32) {
	err := f.boundCharID.Set(int(charID))
	if err != nil {
		log.Printf("Failed to set char ID: %v", err)
	}
	f.updateMails()
}

func (f *folders) updateMails() {
	charID, err := f.boundCharID.Get()
	if err != nil {
		log.Print("Failed to get character ID")
		return
	}
	status := f.esiApp.statusBar
	go func() {
		err = UpdateMails(int32(charID), status)
		if err != nil {
			status.setText("Failed to fetch mail")
			log.Printf("Failed to update mails for character %d: %v", charID, err)
			return
		}
		f.update(int32(charID))
	}()
}

func (f *folders) update(charID int32) {
	if charID == 0 {
		f.refreshButton.Disable()
	} else {
		f.refreshButton.Enable()
	}
	if err := f.boundCharID.Set(int(charID)); err != nil {
		log.Printf("Failed to set char ID: %v", err)
	}

	var ii []interface{}
	labels, err := storage.FetchAllMailLabels(charID)
	if err != nil {
		log.Printf("Failed to fetch mail labels: %v", err)
	} else {
		if len(labels) > 0 {
			ii = append(ii, labelItem{id: allMailsLabelID, name: "All Mails"})
			for _, l := range labels {
				ii = append(ii, labelItem{id: l.LabelID, name: l.Name})
			}
		}
	}
	f.boundList.Set(ii)
	f.list.Select(0)
	f.list.ScrollToTop()
	f.headers.update(charID, allMailsLabelID)
}

func (e *esiApp) newFolders(headers *headers) *folders {
	list, boundList, boundCharID := makeFolderList(headers)
	f := folders{
		esiApp:      e,
		boundList:   boundList,
		boundCharID: boundCharID,
		headers:     headers,
		list:        list,
	}
	f.addRefreshButton()
	c := container.NewBorder(f.refreshButton, nil, nil, nil, f.list)
	f.content = c
	return &f
}

func makeFolderList(headers *headers) (*widget.List, binding.ExternalUntypedList, binding.ExternalInt) {
	var ii []interface{}
	boundList := binding.BindUntypedList(&ii)

	var charID int
	boundCharID := binding.BindInt(&charID)

	container := widget.NewListWithData(
		boundList,
		func() fyne.CanvasObject {
			return widget.NewLabel("from")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			entry, err := i.(binding.Untyped).Get()
			if err != nil {
				log.Println("Failed to label item")
				return
			}
			w := o.(*widget.Label)
			w.SetText(entry.(labelItem).name)
		})

	container.OnSelected = func(iID widget.ListItemID) {
		d, err := boundList.Get()
		if err != nil {
			log.Println("Failed to char ID item")
			return
		}
		n := d[iID].(labelItem)
		cID, err := boundCharID.Get()
		if err != nil {
			log.Println("Failed to Get item")
			return
		}
		headers.update(int32(cID), n.id)

	}
	return container, boundList, boundCharID
}

// UpdateMails fetches and stores new mails from ESI for a character.
func UpdateMails(characterID int32, status *statusBar) error {
	token, err := storage.FetchToken(characterID)
	if err != nil {
		return err
	}
	status.setText("Checking for new mail for %v", token.Character.Name)
	if err := updateMailLists(token); err != nil {
		return err
	}
	if err := updateMailLabels(token); err != nil {
		return err
	}
	headers, err := fetchMailHeaders(token)
	if err != nil {
		return err
	}
	err = updateMails(token, headers, status)
	if err != nil {
		return err
	}
	return nil
}

func updateMailLabels(token *storage.Token) error {
	if err := ensureFreshToken(token); err != nil {
		return err
	}
	ll, err := esi.FetchMailLabels(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	labels := ll.Labels
	log.Printf("Received %d mail labels from ESI for character %d", len(labels), token.CharacterID)
	for _, o := range labels {
		e := storage.MailLabel{
			CharacterID: token.CharacterID,
			LabelID:     o.LabelID,
			Name:        o.Name,
			Color:       o.Color,
			UnreadCount: o.UnreadCount,
		}
		if err := e.Save(); err != nil {
			return err
		}
	}
	return nil
}

func updateMailLists(token *storage.Token) error {
	if err := ensureFreshToken(token); err != nil {
		return err
	}
	lists, err := esi.FetchMailLists(httpClient, token.CharacterID, token.AccessToken)
	if err != nil {
		return err
	}
	for _, o := range lists {
		e := storage.EveEntity{ID: o.ID, Name: o.Name, Category: "mail_list"}
		if err := e.Save(); err != nil {
			return err
		}
	}
	return nil
}

func fetchMailHeaders(token *storage.Token) ([]esi.MailHeader, error) {
	if err := ensureFreshToken(token); err != nil {
		return nil, err
	}
	headers, err := esi.FetchMailHeaders(httpClient, token.CharacterID, token.AccessToken, maxMails)
	if err != nil {
		return nil, err
	}
	log.Printf("Received %d mail headers from ESI for character %d", len(headers), token.CharacterID)
	return headers, nil
}

func updateMails(token *storage.Token, headers []esi.MailHeader, status *statusBar) error {
	existingIDs, missingIDs, err := determineMailIDs(token.CharacterID, headers)
	if err != nil {
		return err
	}
	newMailsCount := missingIDs.Size()
	if newMailsCount == 0 {
		s := "No new mail"
		status.setText(s)
		log.Print(s)
		return nil
	}

	if err := ensureFreshToken(token); err != nil {
		return err
	}

	var c atomic.Int32
	var wg sync.WaitGroup
	for _, header := range headers {
		if existingIDs.Has(header.ID) {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			entityIDs := set.New[int32]()
			entityIDs.Add(header.FromID)
			for _, r := range header.Recipients {
				entityIDs.Add(r.ID)
			}
			if err := addMissingEveEntities(entityIDs.ToSlice()); err != nil {
				log.Printf("Failed to process mail %d: %v", header.ID, err)
				return
			}

			m, err := esi.FetchMail(httpClient, token.CharacterID, header.ID, token.AccessToken)
			if err != nil {
				log.Printf("Failed to process mail %d: %v", header.ID, err)
				return
			}

			mail := storage.Mail{
				Character: token.Character,
				MailID:    header.ID,
				Subject:   header.Subject,
				Body:      m.Body,
			}

			timestamp, err := time.Parse(time.RFC3339, header.Timestamp)
			if err != nil {
				log.Printf("Failed to parse timestamp for mail %d: %v", header.ID, err)
				return
			}
			mail.TimeStamp = timestamp

			from, err := storage.GetEveEntity(header.FromID)
			if err != nil {
				log.Printf("Failed to parse \"from\" mail %d: %v", header.FromID, err)
				return
			}
			mail.From = *from

			var rr []storage.EveEntity
			for _, r := range header.Recipients {
				o, err := storage.GetEveEntity(r.ID)
				if err != nil {
					log.Printf("Failed to resolve recipient %v for mail %d", r, header.ID)
					continue
				} else {
					rr = append(rr, *o)
				}
			}
			mail.Recipients = rr

			labels, err := storage.FetchMailLabels(token.CharacterID, m.Labels)
			if err != nil {
				log.Printf("Failed to resolve labels for mail %d: %v", header.ID, err)
			} else {
				mail.Labels = labels
			}

			mail.Save()
			log.Printf("Stored new mail %d for character %v", header.ID, token.CharacterID)
			c.Add(1)
			current := c.Load()
			status.setText("Fetched %d / %d new mails for %v", current, newMailsCount, token.Character.Name)
		}()
	}
	wg.Wait()
	total := c.Load()
	if total == 0 {
		status.clear()
		return nil
	}
	s := fmt.Sprintf("Stored %d new mails", total)
	status.setText(s)
	log.Print(s)
	return nil
}

func determineMailIDs(characterID int32, headers []esi.MailHeader) (*set.Set[int32], *set.Set[int32], error) {
	ids, err := storage.FetchMailIDs(characterID)
	if err != nil {
		return nil, nil, err
	}
	existingIDs := set.NewFromSlice(ids)
	incomingIDs := set.New[int32]()
	for _, h := range headers {
		incomingIDs.Add(h.ID)
	}
	missingIDs := incomingIDs.Difference(existingIDs)
	return existingIDs, missingIDs, nil
}

// ensureFreshToken will automatically try to refresh a token that is already or about to become invalid.
func ensureFreshToken(token *storage.Token) error {
	if !token.RemainsValid(time.Second * 60) {
		log.Printf("Need to refresh token: %v", token)
		rawToken, err := sso.RefreshToken(httpClient, token.RefreshToken)
		if err != nil {
			return err
		}
		token.AccessToken = rawToken.AccessToken
		token.RefreshToken = rawToken.RefreshToken
		token.ExpiresAt = rawToken.ExpiresAt
		err = token.Save()
		if err != nil {
			return err
		}
		log.Printf("Refreshed token for %v", token.CharacterID)
	}
	return nil
}

func addMissingEveEntities(ids []int32) error {
	c, err := storage.FetchEntityIDs()
	if err != nil {
		return err
	}
	current := set.NewFromSlice(c)
	incoming := set.NewFromSlice(ids)
	missing := incoming.Difference(current)

	if missing.Size() == 0 {
		return nil
	}

	entities, err := esi.ResolveEntityIDs(httpClient, missing.ToSlice())
	if err != nil {
		return fmt.Errorf("failed to resolve IDs: %v", err)
	}

	for _, entity := range entities {
		e := storage.EveEntity{ID: entity.ID, Category: entity.Category, Name: entity.Name}
		err := e.Save()
		if err != nil {
			return err
		}
	}

	log.Printf("Added %d missing eve entities", len(entities))
	return nil
}
