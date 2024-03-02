package main

import (
	"fmt"
	"log"

	"example/goesi-play/esi"
	"example/goesi-play/sso"
)

func main() {
	fmt.Println("Hello")
	scopes := []string{"esi-characters.read_contacts.v1", "esi-universe.read_structures.v1"}
	token, err := sso.Authenticate(scopes)
	if err != nil {
		log.Fatal(err)
	}
	contacts := esi.FetchContacts(token.CharacterID, token.AccessToken)
	fmt.Println(contacts)
}
