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
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/browser"
)

type key int

const (
	keyCodeVerifier           key = iota
	keyError                  key = iota
	keyState                  key = iota
	keyAuthenticatedCharacter key = iota
)

var (
	ErrAborted             = errors.New("auth process canceled prematurely")
	ErrTokenError          = errors.New("token error")
	ErrMissingRefreshToken = errors.New("missing refresh token")
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

// Authenticate an Eve Online character via SSO and return SSO token.
// The process runs in a newly opened browser tab
func Authenticate(ctx context.Context, client *http.Client, scopes []string) (*Token, error) {
	codeVerifier := generateRandomString(32)
	serverCtx := context.WithValue(ctx, keyCodeVerifier, codeVerifier)

	state := generateRandomString(16)
	serverCtx = context.WithValue(serverCtx, keyState, state)

	serverCtx, cancel := context.WithCancel(serverCtx)

	mux := http.NewServeMux()
	mux.HandleFunc(ssoCallbackPath, func(w http.ResponseWriter, req *http.Request) {
		slog.Info("Received SSO callback request")
		v := req.URL.Query()
		newState := v.Get("state")
		if newState != serverCtx.Value(keyState).(string) {
			http.Error(w, "Invalid state", http.StatusForbidden)
			return
		}
		code := v.Get("code")
		codeVerifier := serverCtx.Value(keyCodeVerifier).(string)
		rawToken, err := retrieveTokenPayload(client, code, codeVerifier)
		if err != nil {
			msg := "Failed to retrieve token payload"
			slog.Error(msg, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		claims, err := validateToken(rawToken.AccessToken)
		if err != nil {
			msg := "Failed to validate token"
			slog.Error(msg, "token", rawToken.AccessToken, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		token, err := newToken(rawToken, claims)
		if err != nil {
			msg := "Failed to construct token"
			slog.Error(msg, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		serverCtx = context.WithValue(serverCtx, keyAuthenticatedCharacter, token)
		fmt.Fprintf(
			w,
			"<p>SSO authentication successful for <b>%s</b>.</p><p>You can close this tab now and return to the app.</p>",
			token.CharacterName,
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
		slog.Info("Web server started", "address", address)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("Web server terminated prematurely", "error", err)
		}
	}()

	<-serverCtx.Done() // wait for the signal to gracefully shutdown the server

	err := server.Shutdown(context.Background())
	if err != nil {
		return nil, err
	}
	slog.Info("Web server stopped")

	errValue := serverCtx.Value(keyError)
	if errValue != nil {
		return nil, errValue.(error)
	}

	token, ok := serverCtx.Value(keyAuthenticatedCharacter).(*Token)
	if !ok {
		return nil, ErrAborted
	}
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
func retrieveTokenPayload(client *http.Client, code, codeVerifier string) (*tokenPayload, error) {
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

	slog.Info("Sending auth request to SSO API")
	resp, err := client.Do(req)
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

	slog.Debug("Response from API", "body", string(body))

	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("details %v, %v: %w", token.Error, token.ErrorDescription, ErrTokenError)
	}
	return &token, nil
}

// Update given token with new instance from SSO API
func RefreshToken(client *http.Client, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, ErrMissingRefreshToken
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {ssoClientId},
	}
	rawToken, err := fetchOauthToken(client, form)
	if err != nil {
		return nil, err
	}
	token := Token{
		AccessToken:  rawToken.AccessToken,
		RefreshToken: rawToken.RefreshToken,
		ExpiresAt:    calcExpiresAt(rawToken),
	}
	return &token, nil
}

func fetchOauthToken(client *http.Client, form url.Values) (*tokenPayload, error) {
	req, err := http.NewRequest(
		"POST", ssoTokenUrl, strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", "login.eveonline.com")

	slog.Info("Requesting token from SSO API", "grant_type", form.Get("grant_type"), "url", ssoTokenUrl)
	slog.Debug("Request", "form", form)

	resp, err := client.Do(req)
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

	slog.Debug("Response from SSO API", "body", string(body))

	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("details %v, %v: %w", token.Error, token.ErrorDescription, ErrTokenError)
	}
	return &token, nil
}
