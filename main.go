package main

import (
	"fmt"

	"example/goesi-play/esi"
	"example/goesi-play/sso"
)

func main() {
	fmt.Println("Hello")
	token := sso.Authenticate()
	contacts := esi.FetchContacts(token.CharacterID, token.AccessToken)
	fmt.Println(contacts)
}
