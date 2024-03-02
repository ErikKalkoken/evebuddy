package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	address         = "http://127.0.0.1"
	port            = ":8000"
	siteUrl         = address + port
	ssoClientId     = "882b6f0cbd4e44ad93aead900d07219b"
	ssoClientSecret = "DtCjMrMyoGfqq9TLXCbcJU90aEKEKFCMVWLloYaz"
	ssoCallbackPath = "/sso/callback"
	ssoScopes       = "esi-characters.read_contacts.v1"
	esiBaseUrl      = "https://esi.evetech.net/latest"
)

func init() {
	state = generateState()
}

var state string

type tokenPayload struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int32  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
}

type characterContact struct {
	ContactId   int32   `json:"contact_id"`
	ContactType string  `json:"contact_type"`
	Standing    float32 `json:"standing"`
}

func main() {
	http.HandleFunc("/", index)
	http.HandleFunc(ssoCallbackPath, ssoCallback)
	fmt.Printf("Running server at %v\n", siteUrl)
	http.ListenAndServe(port, nil)
}

func index(w http.ResponseWriter, req *http.Request) {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("redirect_uri", siteUrl+ssoCallbackPath)
	v.Set("client_id", ssoClientId)
	v.Set("state", state)
	v.Set("scope", ssoScopes)

	url := fmt.Sprintf("https://login.eveonline.com/v2/oauth/authorize/?%v", v.Encode())
	fmt.Fprintf(w, "<a href=\"%v\">login</a>\n", url)
}

func ssoCallback(w http.ResponseWriter, req *http.Request) {
	log.Print("Received SSO callback")
	v := req.URL.Query()
	newState := v.Get("state")
	if newState != state {
		log.Fatal("Wrong state")
	}

	code := v.Get("code")

	httpClient := &http.Client{}

	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}}
	req, err := http.NewRequest(
		"POST",
		"https://login.eveonline.com/v2/oauth/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		log.Fatal(err)
	}
	encoded := base64.URLEncoding.EncodeToString([]byte(ssoClientId + ":" + ssoClientSecret))
	req.Header.Add("Authorization", "Basic "+encoded)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	log.Print("Sending auth request to SSO API")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	tokenObj := tokenPayload{}
	jsonErr := json.Unmarshal(body, &tokenObj)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}
	log.Printf("Received token payload from ESI: %v", tokenObj)

	// make authenticated ESI request
	v = url.Values{}
	v.Set("token", tokenObj.AccessToken)
	fullUrl := fmt.Sprintf("%s/characters/%d/contacts/?%v", esiBaseUrl, 93330670, v.Encode())
	log.Printf("Fetching contacts from %v", fullUrl)
	resp, err = httpClient.Get(fullUrl)
	if err != nil {
		log.Fatal(err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var contacts []characterContact
	jsonErr = json.Unmarshal(body, &contacts)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	fmt.Printf("%v", contacts)
}

// Generate a random state string
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	return state
}
