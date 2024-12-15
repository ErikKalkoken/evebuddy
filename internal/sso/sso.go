// Package sso provides the ability to authenticate characters with the Eve Online SSO API for desktop apps.
// It implements OAuth 2.0 with the PKCE protocol.
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

type contextKey int

const (
	keyCodeVerifier contextKey = iota
	keyError
	keyState
	keyAuthenticatedCharacter
)

const (
	portDefault            = 30123
	callbackPathDefault    = "/callback"
	host                   = "localhost"
	ssoHost                = "login.eveonline.com"
	ssoTokenURLDefault     = "https://login.eveonline.com/v2/oauth/token"
	ssoAuthorizeURLDefault = "https://login.eveonline.com/v2/oauth/authorize"
)

var (
	ErrAborted             = errors.New("auth process canceled prematurely")
	ErrTokenError          = errors.New("token error")
	ErrMissingRefreshToken = errors.New("missing refresh token")
)

// SSOService is a service for authentication Eve Online characters.
type SSOService struct {
	// SSO configuration can be modified before calling any method.
	CallbackPath    string
	OAuthURL        string
	Port            int
	SSOAuthorizeURL string
	SSOTokenURL     string

	clientID   string
	httpClient *http.Client
}

// Returns a new SSO service.
func New(clientID string, client *http.Client) *SSOService {
	s := &SSOService{
		CallbackPath:    callbackPathDefault,
		Port:            portDefault,
		SSOAuthorizeURL: ssoAuthorizeURLDefault,
		SSOTokenURL:     ssoTokenURLDefault,

		httpClient: client,
		clientID:   clientID,
	}
	return s
}

// Authenticate an Eve Online character via OAuth 2.0 PKCE and return the new SSO token.
// Will open a new browser tab on the desktop and run a web server for the OAuth process.
func (s *SSOService) Authenticate(ctx context.Context, scopes []string) (*Token, error) {
	codeVerifier, err := generateRandomStringBase64(32)
	if err != nil {
		return nil, err
	}
	serverCtx := context.WithValue(ctx, keyCodeVerifier, codeVerifier)
	state, err := generateRandomStringBase64(16)
	if err != nil {
		return nil, err
	}
	serverCtx = context.WithValue(serverCtx, keyState, state)
	serverCtx, cancel := context.WithCancel(serverCtx)

	mux := http.NewServeMux()
	mux.HandleFunc(s.CallbackPath, func(w http.ResponseWriter, req *http.Request) {
		slog.Info("Received SSO callback request")
		v := req.URL.Query()
		newState := v.Get("state")
		if newState != serverCtx.Value(keyState).(string) {
			err = fmt.Errorf("invalid state")
			msg := "Failed to verify SSO session"
			slog.Warn(msg, "error", err)
			http.Error(w, msg, http.StatusForbidden)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		code := v.Get("code")
		codeVerifier := serverCtx.Value(keyCodeVerifier).(string)
		rawToken, err := s.fetchNewToken(code, codeVerifier)
		if err != nil {
			msg := "Failed to retrieve token payload"
			slog.Warn(msg, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		jwtToken, err := validateJWT(ctx, rawToken.AccessToken)
		if err != nil {
			msg := "Failed to validate token"
			slog.Warn(msg, "token", rawToken.AccessToken, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		characterID, err := extractCharacterID(jwtToken)
		if err != nil {
			msg := "Failed to validate token"
			slog.Warn(msg, "token", rawToken.AccessToken, "error", err)
			http.Error(w, msg, http.StatusInternalServerError)
			serverCtx = context.WithValue(serverCtx, keyError, err)
			cancel()
			return
		}
		characterName := extractCharacterName(jwtToken)
		scopes := extractScopes(jwtToken)
		token := newToken(rawToken, characterID, characterName, scopes)
		serverCtx = context.WithValue(serverCtx, keyAuthenticatedCharacter, token)
		fmt.Fprintf(
			w,
			"<p>SSO authentication successful for <b>%s</b>.</p>"+
				"<p>You can close this tab now and return to the app.</p>",
			token.CharacterName,
		)
		cancel() // shutdown http server
	})

	if err := s.startSSO(state, codeVerifier, scopes); err != nil {
		return nil, err
	}
	server := &http.Server{
		Addr:    s.address(),
		Handler: mux,
	}
	go func() {
		slog.Info("Web server started", "address", s.address())
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("Web server terminated prematurely", "error", err)
		}
	}()

	<-serverCtx.Done() // wait for the signal to gracefully shutdown the server

	if err := server.Shutdown(context.TODO()); err != nil {
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

func (s *SSOService) address() string {
	return fmt.Sprintf("%s:%d", host, s.Port)
}

func (s *SSOService) redirectURI() string {
	return fmt.Sprintf("http://%s%s", s.address(), s.CallbackPath)
}

// Open browser and show character selection for SSO.
func (s *SSOService) startSSO(state string, codeVerifier string, scopes []string) error {
	challenge, err := calcCodeChallenge(codeVerifier)
	if err != nil {
		return err
	}
	u := s.makeStartURL(challenge, state, scopes)
	return browser.OpenURL(u)
}

func (s *SSOService) makeStartURL(challenge, state string, scopes []string) string {
	v := url.Values{}
	v.Set("client_id", s.clientID)
	v.Set("code_challenge_method", "S256")
	v.Set("code_challenge", challenge)
	v.Set("redirect_uri", s.redirectURI())
	v.Set("response_type", "code")
	v.Set("scope", strings.Join(scopes, " "))
	v.Set("state", state)
	return s.SSOAuthorizeURL + "/?" + v.Encode()
}

// fetchNewToken returns a new token from SSO API.
func (s *SSOService) fetchNewToken(code, codeVerifier string) (*tokenPayload, error) {
	form := url.Values{
		"client_id":     {s.clientID},
		"code_verifier": {codeVerifier},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequest("POST", s.SSOTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", ssoHost)

	slog.Info("Sending auth request to SSO API")
	resp, err := s.httpClient.Do(req)
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
	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("retrieve token payload: %s, %s: %w", token.Error, token.ErrorDescription, ErrTokenError)
	}
	return &token, nil
}

// Update given token with new instance from SSO API
func (s *SSOService) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	if refreshToken == "" {
		return nil, ErrMissingRefreshToken
	}
	rawToken, err := s.fetchRefreshedToken(refreshToken)
	if err != nil {
		return nil, err
	}
	_, err = validateJWT(ctx, rawToken.AccessToken)
	if err != nil {
		return nil, err
	}
	token := Token{
		AccessToken:  rawToken.AccessToken,
		RefreshToken: rawToken.RefreshToken,
		ExpiresAt:    rawToken.expiresAt(),
	}
	return &token, nil
}

func (s *SSOService) fetchRefreshedToken(refreshToken string) (*tokenPayload, error) {
	form := url.Values{
		"client_id":     {s.clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	req, err := http.NewRequest("POST", s.SSOTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", ssoHost)
	slog.Info("Requesting token from SSO API", "grant_type", form.Get("grant_type"), "url", s.SSOTokenURL)

	resp, err := s.httpClient.Do(req)
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

	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		return nil, fmt.Errorf("refresh token: %s, %s: %w", token.Error, token.ErrorDescription, ErrTokenError)
	}
	return &token, nil
}

func calcCodeChallenge(codeVerifier string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(codeVerifier)); err != nil {
		return "", err
	}
	challenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return challenge, nil
}

// generateRandomStringBase64 returns a random string of given length with base64 encoding.
func generateRandomStringBase64(length int) (string, error) {
	data := make([]byte, length)
	_, err := rand.Read(data)
	if err != nil {
		return "", err
	}
	s := base64.URLEncoding.EncodeToString(data)
	return s, nil
}
