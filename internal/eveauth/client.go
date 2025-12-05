// Package eveauth provides users the ability to authenticate characters
// with the Eve Online Single Sign-On (SSO) service.
//
// It implements OAuth 2.0 with the PKCS authorization flow
// and is designed primarily for desktop and mobile applications.
package eveauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type contextKey int

const (
	keyCodeVerifier contextKey = iota
	keyState
)

const (
	authorizeURLDefault = "https://login.eveonline.com/v2/oauth/authorize"
	callbackPathDefault = "callback"
	pingTimeout         = 5 * time.Second
	protocol            = "http://"
	resourceHost        = "login.eveonline.com"
	tokenURLDefault     = "https://login.eveonline.com/v2/oauth/token"
)

//go:embed tmpl/*
var templFS embed.FS

var (
	ErrAborted        = errors.New("process aborted prematurely")
	ErrAlreadyRunning = errors.New("another instance is already running")
	ErrInvalid        = errors.New("invalid operation")
	ErrTokenError     = errors.New("token error")
)

// Token represents an OAuth2 token for a character in Eve Online.
type Token struct {
	AccessToken   string
	CharacterID   int32
	CharacterName string
	ExpiresAt     time.Time
	RefreshToken  string
	Scopes        []string
	TokenType     string
}

// newToken creates e new [Token] from a [tokenPayload] and returns it.
func newToken(rawToken *tokenPayload, characterID int, characterName string, scopes []string) *Token {
	t := &Token{
		AccessToken:   rawToken.AccessToken,
		CharacterID:   int32(characterID),
		CharacterName: characterName,
		ExpiresAt:     rawToken.expiresAt(),
		RefreshToken:  rawToken.RefreshToken,
		TokenType:     rawToken.TokenType,
		Scopes:        scopes,
	}
	return t
}

// Config represents the configuration for a client.
type Config struct {
	// The SSO client ID of the Eve Online app. This field is required.
	ClientID string

	// The port for the local webserver to run. This field is required.
	Port int

	// A function to open an URL in the system's browser. This field is required.
	OpenURL func(*url.URL) error

	// The local path for the OAuth2 callback.
	// The default is "callback".
	CallbackPath string

	// The HTTP client to use for all requests. Uses the [http.DefaultClient] by default.
	HTTPClient *http.Client

	// Customer logger instance. Uses slog by default.
	Logger LeveledLogger

	// When enabled will keep the SSO server running and not start the authentication.
	// This feature is for testing purposes only.
	IsDemoMode bool

	// OAuth2 authorization endpoint
	AuthorizeURL string

	// OAuth2 token endpoint
	TokenURL string
}

// client represents a client for authenticating Eve Online characters with the SSO service.
// It is designed for desktop and mobile apps
// and implements OAuth 2.0 with the PKCE protocol.
//
// A client instance is re-usable and applications usually only need to hold one instance.
type client struct {
	authorizeURL     string
	callbackPath     string
	clientID         string
	httpClient       *http.Client
	isAuthenticating atomic.Bool
	isDemoMode       bool
	logger           LeveledLogger
	openURL          func(*url.URL) error
	port             int
	tokenURL         string
}

// NewClient returns a new client for authenticating characters.
//
// A client needs to be configured with config.
// NewClient will return an error if the configuration is invalid.
//
// The callback URL is generated from the configuration and might look like this:
// http://localhost:8000/callback
func NewClient(config Config) (*client, error) {
	if config.ClientID == "" {
		return nil, fmt.Errorf("must specify client ID: %w", ErrInvalid)
	}
	if config.OpenURL == nil {
		return nil, fmt.Errorf("must specify OpenURL: %w", ErrInvalid)
	}
	if config.Port == 0 {
		return nil, fmt.Errorf("must specify port: %w", ErrInvalid)
	}
	s := &client{
		authorizeURL: authorizeURLDefault,
		callbackPath: callbackPathDefault,
		clientID:     config.ClientID,
		httpClient:   http.DefaultClient,
		logger:       slog.Default(),
		openURL:      config.OpenURL,
		port:         config.Port,
		tokenURL:     tokenURLDefault,
	}
	if config.AuthorizeURL != "" {
		s.authorizeURL = config.AuthorizeURL
	}
	if config.CallbackPath != "" {
		cb, _ := strings.CutPrefix(config.CallbackPath, "/")
		s.callbackPath = cb
	}
	if config.HTTPClient != nil {
		s.httpClient = config.HTTPClient
	}
	if config.IsDemoMode {
		s.isDemoMode = config.IsDemoMode
	}
	if config.TokenURL != "" {
		s.tokenURL = config.TokenURL
	}
	if config.Logger != nil {
		s.logger = config.Logger
	}
	return s, nil
}

// Authenticate starts the SSO authentication process for a character
// and returns the new token when successful.
//
// At the beginning the SSO login page will be opened in the browser.
// When completed successfully a landing page will be shown in the browser.
//
// The process can be canceled through ctx and will then return [ErrAborted].
//
// Only one instance of this function may run at the same time.
// Trying to run another instance will return [ErrAlreadyRunning].
//
// Authenticate will temporarily run a local web server with logging enabled.
func (s *client) Authenticate(ctx context.Context, scopes []string) (*Token, error) {
	if !s.isAuthenticating.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("sso-authenticate: %w", ErrAlreadyRunning)
	}
	defer func() {
		s.isAuthenticating.Store(false)
	}()
	codeVerifier, err := generateRandomStringBase64(32)
	if err != nil {
		return nil, fmt.Errorf("sso-authenticate: %w", err)
	}
	serverCtx := context.WithValue(ctx, keyCodeVerifier, codeVerifier)
	state, err := generateRandomStringBase64(16)
	if err != nil {
		return nil, fmt.Errorf("sso-authenticate: %w", err)
	}
	serverCtx = context.WithValue(serverCtx, keyState, state)
	serverCtx, cancel := context.WithCancel(serverCtx)
	defer cancel()

	// result variables. These are returned to caller.
	var (
		errValue atomic.Value
		token    atomic.Pointer[Token]
	)

	processError := func(w http.ResponseWriter, status int, err error) {
		s.logger.Warn("SSO authentication failed", "error", err)
		http.Error(w, fmt.Sprintf("SSO authentication failed: %s", err), status)
		errValue.Store(fmt.Errorf("sso-authenticate: %w", err))
		cancel() // shutdown http server
	}

	router := http.NewServeMux()
	// Route for responding to ping requests
	router.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "pong\n")
	})
	// Route for stopping the server
	router.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		cancel()
	})
	// Route for responding to SSO callback from CCP server
	router.HandleFunc("/"+s.callbackPath, func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query()
		stateGot := v.Get("state")
		stateWant := serverCtx.Value(keyState).(string)
		if stateGot != stateWant {
			processError(w, http.StatusUnauthorized, fmt.Errorf("invalid state. Want: %s - Got: %s", stateWant, stateGot))
			return
		}
		code := v.Get("code")
		codeVerifier := serverCtx.Value(keyCodeVerifier).(string)
		rawToken, err := s.fetchNewToken(code, codeVerifier)
		if err != nil {
			processError(w, http.StatusUnauthorized, fmt.Errorf("fetch new token: %w", err))
			return
		}
		jwtToken, err := validateJWT(ctx, s.httpClient, rawToken.AccessToken)
		if err != nil {
			processError(w, http.StatusUnauthorized, fmt.Errorf("token validation: %w", err))
			return
		}
		characterID, err := extractCharacterID(jwtToken)
		if err != nil {
			processError(w, http.StatusInternalServerError, fmt.Errorf("extract character ID: %w", err))
			return
		}
		characterName := extractCharacterName(jwtToken)
		scopes := extractScopes(jwtToken)
		tok := newToken(rawToken, characterID, characterName, scopes)
		token.Store(tok)
		s.logger.Info("SSO authentication successful", "characterID", tok.CharacterID, "characterName", tok.CharacterName)
		http.Redirect(w, r, "/authenticated", http.StatusSeeOther)
	})
	router.HandleFunc("/authenticated", func(w http.ResponseWriter, r *http.Request) {
		var name, id string
		tok := token.Load()
		if tok != nil {
			name = tok.CharacterName
			id = strconv.Itoa(int(tok.CharacterID))
		} else {
			name = "?"
			id = "1"
		}
		t, err := template.ParseFS(templFS, "tmpl/authenticated.html")
		if err != nil {
			processError(w, http.StatusInternalServerError, err)
			return
		}
		err = t.Execute(w, map[string]string{"Name": name, "ID": id})
		if err != nil {
			processError(w, http.StatusInternalServerError, err)
			return
		}
		if s.isDemoMode {
			return
		}
		cancel() // shutdown http server
	})
	// Route for returning 404 on all other paths
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	// we want to be sure the server is running before starting the browser
	// and we want to be able to exit early in case the port is blocked
	server := &http.Server{
		Addr:    s.address(),
		Handler: newRequestLogger(router, s.logger),
	}
	l, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("sso-authenticate: listen on address: %w", err)
	}
	defer func() {
		if err := server.Close(); err != nil {
			s.logger.Error("sso-server: server close", "error", err)
		}
	}()

	s.logger.Info("sso-server started", "address", protocol+server.Addr)

	go func() {
		if err := server.Serve(l); err != http.ErrServerClosed {
			s.logger.Error("sso-server: server terminated prematurely", "error", err)
		}
		cancel()
		s.logger.Info("sso-server stopped")
	}()

	ctxPing, cncl := context.WithTimeout(ctx, pingTimeout)
	defer cncl()

	u, err := url.JoinPath(protocol+server.Addr, "ping")
	if err != nil {
		return nil, fmt.Errorf("sso-authenticate: invalid path: %w", err)
	}
	req, err := http.NewRequestWithContext(ctxPing, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("sso-authenticate: prepare ping: %w", err)
	}
	_, err = s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sso-authenticate: ping: %w", err)
	}

	if !s.isDemoMode {
		if err := s.startSSO(state, codeVerifier, scopes); err != nil {
			return nil, fmt.Errorf("sso-authenticate: start SSO: %w", err)
		}
	}
	<-serverCtx.Done()

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		s.logger.Warn("sso-server: server shutdown", "error", err)
	}

	if x := errValue.Load(); x != nil {
		return nil, x.(error) // we expect this to always be an error
	}

	t := token.Load()
	if t == nil {
		return nil, fmt.Errorf("sso-authenticate: start SSO: %w", ErrAborted)
	}
	return t, nil
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

func (s *client) address() string {
	return fmt.Sprintf("localhost:%d", s.port)
}

// Open browser and show character selection for SSO.
func (s *client) startSSO(state string, codeVerifier string, scopes []string) error {
	challenge, err := calcCodeChallenge(codeVerifier)
	if err != nil {
		return err
	}
	rawURL, err := s.makeStartURL(challenge, state, scopes)
	if err != nil {
		return err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	return s.openURL(u)
}

func calcCodeChallenge(codeVerifier string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(codeVerifier)); err != nil {
		return "", err
	}
	challenge := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return challenge, nil
}

func (s *client) makeStartURL(challenge, state string, scopes []string) (string, error) {
	uri, err := url.JoinPath(protocol+s.address(), s.callbackPath)
	if err != nil {
		return "", err
	}
	v := url.Values{}
	v.Set("client_id", s.clientID)
	v.Set("code_challenge_method", "S256")
	v.Set("code_challenge", challenge)
	v.Set("redirect_uri", uri)
	v.Set("response_type", "code")
	v.Set("scope", strings.Join(scopes, " "))
	v.Set("state", state)
	return s.authorizeURL + "/?" + v.Encode(), nil
}

// token payload as returned from SSO API
type tokenPayload struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	TokenType        string `json:"token_type"`
	RefreshToken     string `json:"refresh_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// expiresAt returns the time when this token will expire.
func (t *tokenPayload) expiresAt() time.Time {
	x := time.Now().Add(time.Second * time.Duration(t.ExpiresIn))
	return x
}

// fetchNewToken returns a new token from SSO API.
func (s *client) fetchNewToken(code, codeVerifier string) (*tokenPayload, error) {
	form := url.Values{
		"client_id":     {s.clientID},
		"code_verifier": {codeVerifier},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequest("POST", s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", resourceHost)

	s.logger.Info("Sending auth request to SSO API")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		err := fmt.Errorf(
			"SSO new token: token payload has error: %s, %s: %w",
			token.Error, token.ErrorDescription,
			ErrTokenError,
		)
		return nil, err
	}
	return &token, nil
}

// RefreshToken refreshes token when successful
// or returns an error when the refresh has failed.
func (s *client) RefreshToken(ctx context.Context, token *Token) error {
	if token == nil || token.RefreshToken == "" {
		return fmt.Errorf("sso-refresh: missing refresh token: %w", ErrTokenError)
	}
	rawToken, err := s.fetchRefreshedToken(token.RefreshToken)
	if err != nil {
		return fmt.Errorf("sso-refresh: %w", err)
	}
	_, err = validateJWT(ctx, s.httpClient, rawToken.AccessToken)
	if err != nil {
		return fmt.Errorf("sso-refresh: %w", err)
	}
	token.AccessToken = rawToken.AccessToken
	token.RefreshToken = rawToken.RefreshToken
	token.ExpiresAt = rawToken.expiresAt()
	return nil
}

func (s *client) fetchRefreshedToken(refreshToken string) (*tokenPayload, error) {
	form := url.Values{
		"client_id":     {s.clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	req, err := http.NewRequest("POST", s.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Host", resourceHost)
	s.logger.Debug("Requesting token from SSO API", "grant_type", form.Get("grant_type"), "url", s.tokenURL)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	token := tokenPayload{}
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, err
	}
	if token.Error != "" {
		err := fmt.Errorf(
			"SSO refresh token: token payload has error: %s, %s: %w",
			token.Error,
			token.ErrorDescription,
			ErrTokenError,
		)
		return nil, err
	}
	return &token, nil
}
