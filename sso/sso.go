package sso

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
	oauthURL        = "https://login.eveonline.com/.well-known/oauth-authorization-server"
)

type key int

const (
	keyToken key = iota
	keyState key = iota
)

type Token struct {
	AccessToken   string `json:"access_token"`
	ExpiresIn     int32  `json:"expires_in"`
	TokenType     string `json:"token_type"`
	RefreshToken  string `json:"refresh_token"`
	CharacterID   int32
	CharacterName string
	Scopes        []string
}

// Authenticate an Eve Online character via SSO
// Returns an SSO token and an error
func Authenticate(scopes []string) (*Token, error) {
	state := generateState()
	ctx := context.WithValue(context.Background(), keyState, state)
	ctx, cancel := context.WithCancel(ctx)

	http.HandleFunc(ssoCallbackPath, func(w http.ResponseWriter, req *http.Request) {
		log.Print("Received SSO callback")
		v := req.URL.Query()
		newState := v.Get("state")
		if newState != ctx.Value(keyState).(string) {
			http.Error(w, "Invalid state", http.StatusForbidden)
			return
		}

		code := v.Get("code")
		token, err := retrieveToken(code)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		claims, err := validateToken(token.AccessToken)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		characterID, err := extractCharacterID(claims)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		token.CharacterID = int32(characterID)
		token.CharacterName = claims["name"].(string)

		var scopes []string
		for _, v := range claims["scp"].([]interface{}) {
			s := v.(string)
			scopes = append(scopes, s)
		}

		token.Scopes = scopes
		ctx = context.WithValue(ctx, keyToken, token)

		fmt.Fprintf(
			w,
			"Authentication completed for %s. Scopes granted: %s. You can close this window now.",
			token.CharacterName,
			strings.Join(scopes, ", "),
		)
		cancel() // shutdown http server
	})

	if err := startSSO(state, scopes); err != nil {
		return nil, err
	}
	server := &http.Server{
		Addr: address,
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

	token := ctx.Value(keyToken).(*Token)
	return token, nil
}

// Open browser and show character selection for SSO
func startSSO(state string, scopes []string) error {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("redirect_uri", "http://"+address+ssoCallbackPath)
	v.Set("client_id", ssoClientId)
	v.Set("state", state)
	v.Set("scope", strings.Join(scopes, " "))

	url := fmt.Sprintf("https://login.eveonline.com/v2/oauth/authorize/?%v", v.Encode())
	err := browser.OpenURL(url)
	return err
}

// Retrieve SSO token from API in exchange for code
func retrieveToken(code string) (*Token, error) {
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
		return nil, err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, err
	}

	token := Token{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// Generate a random state string
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	return state
}

// Validate SSO token and return claims
func validateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, getKey)
	if err != nil {
		return nil, err
	}
	claims := token.Claims.(jwt.MapClaims)
	iss := claims["iss"].(string)
	if iss != "login.eveonline.com" && iss != "https://login.eveonline.com" {
		return nil, fmt.Errorf("invalid issuer claim")
	}
	return claims, nil
}

// Return public key for JWT token
func getKey(token *jwt.Token) (interface{}, error) {
	jwksURL, err := determineJwksURL()
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

// Determine URL for JWK sets dynamically from web site and return it
func determineJwksURL() (string, error) {
	resp, err := http.Get(oauthURL)
	if err != nil {
		return "", err
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	jwksURL := data["jwks_uri"].(string)
	return jwksURL, nil
}

// Extract character ID from JWT claims
func extractCharacterID(claims jwt.MapClaims) (int, error) {
	characterID, err := strconv.Atoi(strings.Split(claims["sub"].(string), ":")[2])
	return characterID, err
}
