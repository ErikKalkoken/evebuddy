package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/pkg/browser"

	"example/esiapp/internal/storage"
)

const (
	host            = "127.0.0.1"
	port            = ":8000"
	address         = host + port
	ssoClientId     = "882b6f0cbd4e44ad93aead900d07219b"
	ssoCallbackPath = "/sso/callback"
	oauthURL        = "https://login.eveonline.com/.well-known/oauth-authorization-server"
	ssoIssuer1      = "login.eveonline.com"
	ssoIssuer2      = "https://login.eveonline.com"
)

type key int

const (
	keyCodeVerifier key = iota
	keyError        key = iota
	keyState        key = iota
	keyToken        key = iota
)

type tokenPayload struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int32  `json:"expires_in"`
	TokenType        string `json:"token_type"`
	RefreshToken     string `json:"refresh_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// Authenticate an Eve Online character via SSO and return SSO token.
// The process runs in a newly opened browser tab
func Authenticate(scopes []string) (*storage.Token, error) {
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

		token, err := buildToken(rawToken, claims)
		if err != nil {
			log.Printf("Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ctx = context.WithValue(ctx, keyError, err)
			cancel()
			return
		}
		ctx = context.WithValue(ctx, keyToken, token)

		fmt.Fprintf(
			w,
			"Authentication completed for %s. Scopes granted: %s. You can close this window now.",
			token.CharacterName,
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

	token := ctx.Value(keyToken).(*storage.Token)
	return token, nil
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

// Validate JWT token and return claims
func validateToken(tokenString string) (jwt.MapClaims, error) {
	// parse token and validate signature
	token, err := jwt.Parse(tokenString, getKey)
	if err != nil {
		return nil, err
	}

	// validate issuer claim
	claims := token.Claims.(jwt.MapClaims)
	iss := claims["iss"].(string)
	if iss != ssoIssuer1 && iss != ssoIssuer2 {
		return nil, fmt.Errorf("invalid issuer claim")
	}

	// validate audience claim
	aud := claims["aud"].([]interface{})
	if aud[0].(string) != ssoClientId {
		return nil, fmt.Errorf("invalid first audience claim")
	}
	if aud[1].(string) != "EVE Online" {
		return nil, fmt.Errorf("invalid 2nd audience claim")
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

// build storage.Token object
func buildToken(rawToken *tokenPayload, claims jwt.MapClaims) (*storage.Token, error) {
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

	// calc expires at
	expiresAt := time.Now().Add(time.Second * time.Duration(rawToken.ExpiresIn))

	token := storage.Token{
		AccessToken:   rawToken.AccessToken,
		CharacterID:   int32(characterID),
		CharacterName: claims["name"].(string),
		ExpiresAt:     expiresAt,
		RefreshToken:  rawToken.RefreshToken,
		TokenType:     rawToken.TokenType,
	}

	return &token, nil
}