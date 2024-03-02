package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/pkg/browser"
)

const (
	host            = "127.0.0.1"
	port            = ":8000"
	address         = host + port
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
	ctx, cancel := context.WithCancel(context.Background())

	http.HandleFunc(ssoCallbackPath, func(w http.ResponseWriter, req *http.Request) {
		log.Print("Received SSO callback")
		v := req.URL.Query()
		newState := v.Get("state")
		if newState != state {
			log.Fatal("Wrong state")
		}

		code := v.Get("code")
		tokenObj := retrieveToken(code)
		fmt.Fprintf(w, "Authentication completed. You can close this window now.")

		cancel() // shutdown http server

		claims := validateToken(tokenObj.AccessToken)
		characterID, err := extractCharacterID(claims)
		if err != nil {
			log.Fatal(err)
		}

		// make authenticated ESI request
		contacts := fetchContacts(characterID, tokenObj.AccessToken)
		fmt.Printf("%v", contacts)
	})

	server := &http.Server{
		Addr: address,
	}
	fmt.Printf("Running server at %v\n", address)
	startSSO()
	go func() {
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Println(err)
		}
	}()

	<-ctx.Done() // wait for the signal to gracefully shutdown the server

	// gracefully shutdown the server:
	// waiting indefinitely for connections to return to idle and then shut down.
	err := server.Shutdown(context.Background())
	if err != nil {
		log.Println(err)
	}

	log.Println("done.")
}

func extractCharacterID(claims jwt.MapClaims) (int, error) {
	characterID, err := strconv.Atoi(strings.Split(claims["sub"].(string), ":")[2])
	return characterID, err
}

func retrieveToken(code string) tokenPayload {
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
	httpClient := &http.Client{}
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
	if err := json.Unmarshal(body, &tokenObj); err != nil {
		log.Fatal(err)
	}
	return tokenObj
}

func startSSO() {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("redirect_uri", "http://"+address+ssoCallbackPath)
	v.Set("client_id", ssoClientId)
	v.Set("state", state)
	v.Set("scope", ssoScopes)

	url := fmt.Sprintf("https://login.eveonline.com/v2/oauth/authorize/?%v", v.Encode())
	err := browser.OpenURL(url)
	if err != nil {
		log.Fatal(err)
	}
}

// Generate a random state string
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	return state
}

func validateToken(tokenString string) jwt.MapClaims {
	token, err := jwt.Parse(tokenString, getKey)
	if err != nil {
		panic(err)
	}
	claims := token.Claims.(jwt.MapClaims)
	return claims
}

func fetchJson(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

func getKey(token *jwt.Token) (interface{}, error) {
	jwksURL, err := fetchJwksURL()
	if err != nil {
		return nil, err
	}
	// TODO: cache response so we don't have to make a request every time
	// we want to verify a JWT
	set, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		return nil, err
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("expecting JWT header to have string kid")
	}

	key, ok := set.LookupKeyID(keyID)
	if !ok {
		return nil, fmt.Errorf("unable to find key %q", keyID)
	}

	var rawKey interface{}
	if err := key.Raw(&rawKey); err != nil {
		return nil, fmt.Errorf("failed to create public key: %s", err)
	}
	return rawKey, nil
}

func fetchJwksURL() (string, error) {
	const url = "https://login.eveonline.com/.well-known/oauth-authorization-server"
	body := fetchJson(url)
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	jwksURL := data["jwks_uri"].(string)
	return jwksURL, nil
}

func fetchContacts(characterID int, tokenString string) []characterContact {
	v := url.Values{}
	v.Set("token", tokenString)
	fullUrl := fmt.Sprintf("%s/characters/%d/contacts/?%v", esiBaseUrl, characterID, v.Encode())
	log.Printf("Fetching contacts from %v", fullUrl)
	resp, err := http.Get(fullUrl)
	if err != nil {
		log.Fatal(err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var contacts []characterContact
	if err := json.Unmarshal(body, &contacts); err != nil {
		log.Fatal(err)
	}
	return contacts
}
