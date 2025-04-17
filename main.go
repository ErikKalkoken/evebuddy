// Evebuddy is a companion app for Eve Online players.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"maps"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/antihax/goesi"
	"github.com/chasinglogic/appdirs"
	"github.com/gohugoio/httpcache"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/juju/mutex/v2"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/esistatusservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/evenotification"
	"github.com/ErikKalkoken/evebuddy/internal/app/eveuniverseservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/pcache"
	"github.com/ErikKalkoken/evebuddy/internal/app/settings"
	"github.com/ErikKalkoken/evebuddy/internal/app/statuscacheservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/storage"
	"github.com/ErikKalkoken/evebuddy/internal/app/ui"
	"github.com/ErikKalkoken/evebuddy/internal/deleteapp"
	"github.com/ErikKalkoken/evebuddy/internal/eveimageservice"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/ErikKalkoken/evebuddy/internal/sso"
)

const (
	appID               = "io.github.erikkalkoken.evebuddy"
	appName             = "evebuddy"
	cacheCleanUpTimeout = time.Minute * 30
	crashFileName       = "crash.txt"
	dbFileName          = appName + ".sqlite"
	logFileName         = appName + ".log"
	logFolderName       = "log"
	logLevelDefault     = slog.LevelWarn // for startup only
	logMaxBackups       = 3
	logMaxSizeMB        = 50
	mutexDelay          = 100 * time.Millisecond
	mutexTimeout        = 250 * time.Millisecond
	ssoClientID         = "11ae857fe4d149b2be60d875649c05f1"
	userAgent           = "EveBuddy kalkoken87@gmail.com"
)

// Resonses from these URLs will never be logged.
var blacklistedURLs = []string{"login.eveonline.com/v2/oauth/token"}

// define flags
var (
	deleteDataFlag     = flag.Bool("delete-data", false, "Delete user data")
	developFlag        = flag.Bool("dev", false, "Enable developer features")
	dirsFlag           = flag.Bool("dirs", false, "Show directories for user data")
	disableUpdatesFlag = flag.Bool("disable-updates", false, "Disable all periodic updates")
	logLevelFlag       = flag.String("log-level", "", "Set log level for this session")
	offlineFlag        = flag.Bool("offline", false, "Start app in offline mode")
	pprofFlag          = flag.Bool("pprof", false, "Enable pprof web server")
	resetSettingsFlag  = flag.Bool("reset-settings", false, "Resets desktop settings")
	versionFlag        = flag.Bool("v", false, "Show version")
	ssoDemoFlag        = flag.Bool("sso-demo", false, "Start SSO serer in demo mode")
)

func main() {
	// init log & flags
	slog.SetLogLoggerLevel(logLevelDefault)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	flag.Parse()

	// set manual log level for this session if requested
	if v := *logLevelFlag; v != "" {
		m := map[string]slog.Level{
			"debug": slog.LevelDebug,
			"info":  slog.LevelInfo,
			"warn":  slog.LevelWarn,
			"error": slog.LevelError,
		}
		l, ok := m[strings.ToLower(v)]
		if !ok {
			fmt.Println("valid log levels are: ", strings.Join(slices.Collect(maps.Keys(m)), ", "))
			os.Exit(1)
		}
		slog.SetLogLoggerLevel(l)
	}

	// start fyne app
	fyneApp := app.NewWithID(appID)
	_, isDesktop := fyneApp.(desktop.App)

	if *versionFlag {
		fmt.Println(fyneApp.Metadata().Version)
		return
	}

	// set log level from settings
	if *logLevelFlag == "" {
		s := settings.New(fyneApp.Preferences())
		if l := s.LogLevelSlog(); l != logLevelDefault {
			slog.Info("Setting log level", "level", l)
			slog.SetLogLoggerLevel(l)
		}
	}

	var dataDir string

	// data dir
	if isDesktop || *developFlag {
		ad := appdirs.New(appName)
		dataDir = ad.UserData()
		if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	} else {
		dataDir = fyneApp.Storage().RootURI().Path()
	}

	if *dirsFlag {
		fmt.Println(dataDir)
		fmt.Println(fyneApp.Storage().RootURI().Path())
		return
	}

	// desktop related init
	if isDesktop {
		// start uninstall app if requested
		if *deleteDataFlag {
			u := deleteapp.NewUI(fyneApp)
			u.DataDir = dataDir
			u.ShowAndRun()
			return
		}
	}

	// setup logfile for desktop
	logDir := filepath.Join(dataDir, logFolderName)
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}
	logFilePath := filepath.Join(logDir, logFileName)
	logger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    logMaxSizeMB,
		MaxBackups: logMaxBackups,
	}
	defer logger.Close()
	var logWriter io.Writer
	if runtime.GOOS == "windows" {
		logWriter = logger
	} else {
		logWriter = io.MultiWriter(os.Stderr, logger)
	}
	log.SetOutput(logWriter)

	if *ssoDemoFlag {
		sso := sso.New("", http.DefaultClient)
		sso.DemoMode = true
		sso.Authenticate(context.Background(), []string{})
		return
	}

	if isDesktop {
		// ensure single instance
		mu, err := ensureSingleInstance()
		if err != nil {
			log.Fatal(err)
		}
		defer mu.Release()
	}

	crashFilePath := setupCrashFile(logDir)

	// start pprof web server
	if *pprofFlag {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// init database
	dbPath := filepath.Join(dataDir, dbFileName)
	dsn := "file:///" + filepath.ToSlash(dbPath)
	dbRW, dbRO, err := storage.InitDB(dsn)
	if err != nil {
		slog.Error("Failed to initialize database", "dsn", dsn, "error", err)
		os.Exit(1)
	}
	defer dbRW.Close()
	defer dbRO.Close()
	st := storage.New(dbRW, dbRO)

	// Initialize caches
	memCache := memcache.New()
	defer memCache.Close()
	pc := pcache.New(st, cacheCleanUpTimeout)
	defer pc.Close()

	// Initialize shared HTTP client
	// Automatically retries on connection and most server errors
	// Logs requests on debug level and all HTTP error responses as warnings
	rhc := retryablehttp.NewClient()
	rhc.HTTPClient.Transport = &httpcache.Transport{
		Cache:               newCacheAdapter(pc, "httpcache-", 24*time.Hour),
		MarkCachedResponses: true,
	}
	rhc.Logger = slog.Default()
	rhc.ResponseLogHook = logResponse

	// Initialize shared ESI client
	esiClient := goesi.NewAPIClient(rhc.StandardClient(), userAgent)

	// Init StatusCache service
	scs := statuscacheservice.New(memCache)
	if err := scs.InitCache(context.TODO(), st); err != nil {
		slog.Error("Failed to init cache", "error", err)
		os.Exit(1)
	}
	// Init EveUniverse service
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		Storage:            st,
		ESIClient:          esiClient,
		StatusCacheService: scs,
	})

	// Init EveNotification service
	en := evenotification.New(eus)

	// Init Character service
	ssoService := sso.New(ssoClientID, rhc.StandardClient())
	ssoService.OpenURL = fyneApp.OpenURL
	cs := characterservice.New(characterservice.Params{
		Storage:                st,
		HttpClient:             rhc.StandardClient(),
		EsiClient:              esiClient,
		EveNotificationService: en,
		EveUniverseService:     eus,
		StatusCacheService:     scs,
		SSOService:             ssoService,
	})

	// Init UI
	bu := ui.NewBaseUI(ui.BaseUIParams{
		App:                fyneApp,
		CharacterService:   cs,
		EveImageService:    eveimageservice.New(pc, rhc.StandardClient(), *offlineFlag),
		ESIStatusService:   esistatusservice.New(esiClient),
		EveUniverseService: eus,
		StatusCacheService: scs,
		MemCache:           memCache,
		IsOffline:          *offlineFlag,
		IsUpdateDisabled:   *disableUpdatesFlag,
		DataPaths: map[string]string{
			"db":        dbPath,
			"log":       logFilePath,
			"crashfile": crashFilePath,
		},
		ClearCacheFunc: func() {
			pc.Clear()
			memCache.Clear()
		},
	})
	if isDesktop {
		u := ui.NewDesktopUI(bu)
		u.Init()
		if *resetSettingsFlag {
			u.ResetDesktopSettings()
		}
		u.ShowAndRun()
	} else {
		u := ui.NewMobileUI(bu)
		u.Init()
		u.ShowAndRun()
	}
}

type realtime struct{}

func (r realtime) After(d time.Duration) <-chan time.Time {
	c := make(chan time.Time)
	go func() {
		time.Sleep(d)
		c <- time.Now()
	}()
	return c
}

func (r realtime) Now() time.Time {
	return time.Now()
}

// ensureSingleInstance sets and returns a mutex for this application instance.
// The returned mutex must not be released until the application terminates.
func ensureSingleInstance() (mutex.Releaser, error) {
	slog.Debug("Checking for other instances")
	mu, err := mutex.Acquire(mutex.Spec{
		Name:    strings.ReplaceAll(appID, ".", "-"),
		Clock:   realtime{},
		Delay:   mutexDelay,
		Timeout: mutexTimeout,
	})
	if errors.Is(err, mutex.ErrTimeout) {
		return nil, fmt.Errorf("another instance running")
	} else if err != nil {
		return nil, fmt.Errorf("acquire mutex: %w", err)
	}
	slog.Info("No other instances running")
	return mu, nil
}

func setupCrashFile(logDir string) (path string) {
	path = filepath.Join(logDir, crashFileName)
	crashFile, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		slog.Error("Failed to open crash report file", "error", err)
		return
	}
	if err := debug.SetCrashOutput(crashFile, debug.CrashOptions{}); err != nil {
		slog.Error("Failed to setup crash report", "error", err)
	}
	crashFile.Close()
	return
}

// logResponse is a callback for retryable logger, which is called for every respose.
// It logs all HTTP erros and also the complete response when log level is DEBUG.
func logResponse(l retryablehttp.Logger, r *http.Response) {
	isDebug := slog.Default().Enabled(context.Background(), slog.LevelDebug)
	isHttpError := r.StatusCode >= 400
	if !isDebug && !isHttpError {
		return
	}

	var level slog.Level
	if isHttpError {
		level = slog.LevelWarn
	} else {
		level = slog.LevelDebug

	}
	status := statusText(r)
	body := bodyToString(r)
	var args []any
	if isDebug {
		args = []any{
			"method", r.Request.Method,
			"url", r.Request.URL,
			"status", status,
			"header", r.Header,
			"body", body,
		}
	} else {
		args = []any{
			"method", r.Request.Method,
			"url", r.Request.URL,
			"status", status,
			"body", body,
		}
	}

	slog.Log(context.Background(), level, "HTTP response", args...)
}

func bodyToString(r *http.Response) string {
	if r.Body == nil {
		return ""
	}
	hasBlockedURL := slices.ContainsFunc(blacklistedURLs, func(x string) bool {
		return strings.Contains(r.Request.URL.String(), x)
	})
	if hasBlockedURL {
		return "xxxxx"
	}
	var s string
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s = "ERROR: " + err.Error()
	} else {
		s = string(body)
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return s
}

func statusText(r *http.Response) string {
	var s string
	if r.StatusCode == 420 {
		s = "Error Limited"
	} else {
		s = http.StatusText(r.StatusCode)
	}
	return fmt.Sprintf("%d %s", r.StatusCode, s)
}

// cacheAdapter enabled the use of pcache with httpcache
type cacheAdapter struct {
	c       *pcache.PCache
	prefix  string
	timeout time.Duration
}

var _ httpcache.Cache = (*cacheAdapter)(nil)

// newCacheAdapter returns a new cacheAdapter.
// The prefix is added to all cache keys to prevent conflicts.
// Keys are stored with the given cache timeout. A timeout of 0 means that keys never expire.
func newCacheAdapter(c *pcache.PCache, prefix string, timeout time.Duration) *cacheAdapter {
	ca := &cacheAdapter{c: c, prefix: prefix, timeout: timeout}
	return ca
}

func (ca *cacheAdapter) Get(key string) ([]byte, bool) {
	return ca.c.Get(ca.makeKey(key))
}

func (ca *cacheAdapter) Set(key string, b []byte) {
	ca.c.Set(ca.makeKey(key), b, ca.timeout)
}

func (ca *cacheAdapter) Delete(key string) {
	ca.c.Delete(ca.makeKey(key))
}

func (ca *cacheAdapter) makeKey(key string) string {
	return ca.prefix + key
}
