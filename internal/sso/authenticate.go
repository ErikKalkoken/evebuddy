package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/browser"
)

type key int

const (
	keyCodeVerifier           key = iota
	keyError                  key = iota
	keyState                  key = iota
	keyAuthenticatedCharacter key = iota
)

// token payload as returned from SSO API
type tokenPayload struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int32  `json:"expires_in"`
	TokenType        string `json:"token_type"`
	RefreshToken     string `json:"refresh_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// SSO token for Eve Online
type Token struct {
	AccessToken   string
	CharacterID   int32
	CharacterName string
	ExpiresAt     time.Time
	RefreshToken  string
	TokenType     string
}

// Authenticate an Eve Online character via SSO and return SSO token.
// The process runs in a newly opened browser tab
func Authenticate(scopes []string) (*Token, error) {
	codeVerifier := generateRandomString(32)
	ctx := context.WithValue(context.Background(), keyCodeVerifier, codeVerifier)

	state := generateRandomString(16)
	ctx = context.WithValue(ctx, keyState, state)

	ctx, cancel := context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc(ssoCallbackPath, func(w http.ResponseWriter, req *http.Request) {
		log.Print("Received SSO callback")
		v := req.URL.Query()
		newState := v.Get("state")
		if newState != ctx.Value(keyState).(string) {
			http.Error(w, "Invalid state", http.StatusForbidden)
			return
		}

		code := v.Get("code")
		codeVerifier := ctx.Value(keyCodeVerifier).(string)
		rawToken, err := retrieveTokenPayload(code, codeVerifier)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ctx = context.WithValue(ctx, keyError, err)
			cancel()
			return
		}

		claims, err := validateToken(rawToken.AccessToken)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ctx = context.WithValue(ctx, keyError, err)
			cancel()
			return
		}

		character, err := buildToken(rawToken, claims)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ctx = context.WithValue(ctx, keyError, err)
			cancel()
			return
		}
		ctx = context.WithValue(ctx, keyAuthenticatedCharacter, character)

		fmt.Fprintf(
			w,
			"Authentication completed for %s. Scopes granted: %s. You can close this window now.",
			character.CharacterName,
			strings.Join(scopes, ", "),
		)
		cancel() // shutdown http server
	})

	if err := startSSO(state, codeVerifier, scopes); err != nil {
		return nil, err
	}
	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	go func() {
		log.Printf("Web server started at %v\n", address)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Println(err)
		}
	}()

	<-ctx.Done() // wait for the signal to gracefully shutdown the server

	err := server.Shutdown(context.Background())
	if err != nil {
		return nil, err
	}
	log.Println("Web server stopped")

	errValue := ctx.Value(keyError)
	if errValue != nil {
		return nil, errValue.(error)
	}

	character := ctx.Value(keyAuthenticatedCharacter).(*Token)
	return character, nil
}

// Generate a random string of given length
func generateRandomString(length int) string {
	data := make([]byte, length)
	rand.Read(data)
	v := base64.URLEncoding.EncodeToString(data)
	return v
}

// Open browser and show character selection for SSO
func startSSO(state string, codeVerifier string, scopes []string) error {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("redirect_uri", "http://"+address+ssoCallbackPath)
	v.Set("client_id", ssoClientId)
	v.Set("scope", strings.Join(scopes, " "))
	v.Set("state", state)
	v.Set("code_challenge", calcCodeChallenge(codeVerifier))
	v.Set("code_challenge_method", "S256")

	url := fmt.Sprintf("https://login.eveonline.com/v2/oauth/authorize/?%v", v.Encode())
	err := browser.OpenURL(url)
	return err
}

func calcCodeChallenge(codeVerifier string) string {
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	bs := h.Sum(nil)
	challenge := base64.RawURLEncoding.EncodeToString(bs)
	return challenge
}

// Retrieve SSO token from API in exchange for code
func retrieveTokenPayload(code, codeVerifier string) (*tokenPayload, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {ssoClientId},
		"code_verifier": {codeVerifier},
	}
	req, err := http.NewRequest(
		"POST",
		"https://login.eveonline.com/v2/oauth/token",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", ssoHost)

	log.Print("Sending auth request to SSO API")
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, err
	}

	log.Printf("Response from API: %v", string(body))

	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("API response: %v: %v", token.Error, token.ErrorDescription)
	}
	return &token, nil
}

// build storage.Token object
func buildToken(rawToken *tokenPayload, claims jwt.MapClaims) (*Token, error) {
	// calc character ID
	characterID, err := strconv.Atoi(strings.Split(claims["sub"].(string), ":")[2])
	if err != nil {
		return nil, err
	}

	// calc scopes
	// var scopes []string
	// for _, v := range claims["scp"].([]interface{}) {
	// 	s := v.(string)
	// 	scopes = append(scopes, s)
	// }

	token := Token{
		AccessToken:   rawToken.AccessToken,
		CharacterID:   int32(characterID),
		CharacterName: claims["name"].(string),
		ExpiresAt:     calcExpiresAt(rawToken),
		RefreshToken:  rawToken.RefreshToken,
		TokenType:     rawToken.TokenType,
	}

	return &token, nil
}

func calcExpiresAt(rawToken *tokenPayload) time.Time {
	expiresAt := time.Now().Add(time.Second * time.Duration(rawToken.ExpiresIn))
	return expiresAt
}

// Update given token with new instance from SSO API
func RefreshToken(refreshToken string) (*Token, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {ssoClientId},
	}
	rawToken, err := fetchOauthToken(form)
	if err != nil {
		return nil, err
	}
	character := Token{
		AccessToken:  rawToken.AccessToken,
		RefreshToken: rawToken.RefreshToken,
		ExpiresAt:    calcExpiresAt(rawToken),
	}
	return &character, nil
}

func fetchOauthToken(form url.Values) (*tokenPayload, error) {
	req, err := http.NewRequest(
		"POST", ssoTokenUrl, strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", "login.eveonline.com")

	log.Printf("Requesting token from SSO API by %s", form.Get("grant_type"))
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, err
	}

	log.Printf("Response from SSO API: %v", string(body))

	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("SSO API error: %v: %v", token.Error, token.ErrorDescription)
	}
	return &token, nil
}
