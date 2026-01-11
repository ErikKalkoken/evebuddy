// Evebuddy is a companion app for Eve Online players.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"maps"
	"math"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
	"github.com/ErikKalkoken/eveauth"
	"github.com/antihax/goesi"
	"github.com/chasinglogic/appdirs"
	"github.com/gohugoio/httpcache"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/juju/mutex/v2"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ErikKalkoken/evebuddy/internal/app/characterservice"
	"github.com/ErikKalkoken/evebuddy/internal/app/corporationservice"
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
	"github.com/ErikKalkoken/evebuddy/internal/janiceservice"
	"github.com/ErikKalkoken/evebuddy/internal/memcache"
	"github.com/ErikKalkoken/evebuddy/internal/remoteservice"
	"github.com/ErikKalkoken/evebuddy/internal/xgoesi"
	"github.com/ErikKalkoken/evebuddy/internal/xmaps"
)

const (
	appID               = "io.github.erikkalkoken.evebuddy"
	appName             = "evebuddy"
	authClientID        = "11ae857fe4d149b2be60d875649c05f1"
	authPort            = 30123
	remotePort          = 30125
	cacheCleanUpTimeout = time.Minute * 30
	concurrentLimit     = 10 // max concurrent Goroutines per group
	crashFileName       = "crash.txt"
	dbFileName          = appName + ".sqlite"
	logFileName         = appName + ".log"
	logFolderName       = "log"
	logLevelDefault     = slog.LevelWarn // for startup only
	logMaxBackups       = 3
	logMaxSizeMB        = 50
	maxCPUShare         = 0.5
	mutexDelay          = 100 * time.Millisecond
	mutexTimeout        = 250 * time.Millisecond
	sourceURL           = "https://github.com/ErikKalkoken/evebuddy"
	userAgentEmail      = "kalkoken87@gmail.com"
)

// define flags
var (
	clearCacheFlag          = flag.Bool("clear-cache", false, "Clear the cache")
	deleteDataFlag          = flag.Bool("delete-data", false, "Delete user data")
	deleteDataNoConfirmFlag = flag.Bool(
		"delete-data-no-confirm",
		false,
		"Delete user data without asking for confirmation",
	)
	developFlag        = flag.Bool("dev", false, "Enable developer features")
	disableUpdatesFlag = flag.Bool("disable-updates", false, "Disable all periodic updates")
	filesFlag          = flag.Bool("files", false, "Show paths to data files")
	logLevelFlag       = flag.String("log-level", "", "Set log level for this session")
	mobileFlag         = flag.Bool("mobile", false, "Run the app in forced mobile mode")
	offlineFlag        = flag.Bool("offline", false, "Start app in offline mode")
	pprofFlag          = flag.Bool("pprof", false, "Enable pprof web server")
	resetUIFlag        = flag.Bool("reset-ui", false, "Resets UI settings to defaults")
	ssoDemoFlag        = flag.Bool("sso-demo", false, "Start SSO serer in demo mode")
	versionFlag        = flag.Bool("v", false, "Show version")
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

	// Set CPUs utilization limit
	cpuTotal := runtime.NumCPU()
	cpuLimit := max(1, int(math.Floor(float64(cpuTotal)*maxCPUShare)))
	slog.Info("CPUs usage", "total", cpuTotal, "limit", cpuLimit)
	runtime.GOMAXPROCS(cpuLimit)

	// start fyne app
	fyneApp := app.NewWithID(appID)
	if *versionFlag {
		fmt.Println(fyneApp.Metadata().Version)
		return
	}

	// File paths
	var dataDir string
	var isDesktop bool
	if !*mobileFlag {
		_, isDesktop = fyneApp.(desktop.App)
	}
	if isDesktop || *developFlag {
		ad := appdirs.New(appName)
		dataDir = ad.UserData()
		if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	} else {
		dataDir = fyneApp.Storage().RootURI().Path()
	}
	dbPath := filepath.Join(dataDir, dbFileName)
	logDir := filepath.Join(dataDir, logFolderName)
	logFilePath := filepath.Join(logDir, logFileName)
	crashFilePath := filepath.Join(logDir, crashFileName)
	dataPaths := xmaps.OrderedMap[string, string]{
		"db":        dbPath,
		"log":       logFilePath,
		"crashfile": crashFilePath,
		"settings":  path.Join(fyneApp.Storage().RootURI().Path(), "preferences.json"),
	}

	if *filesFlag {
		for k, v := range dataPaths.All() {
			fmt.Printf("%s: %s\n", k, v)
		}
		return
	}

	log.Printf("INFO EVE Buddy version=%s", fyneApp.Metadata().Version)

	appSettings := settings.New(fyneApp.Preferences())
	if *resetUIFlag {
		appSettings.ResetUI()
	}

	// set log level from settings
	if *logLevelFlag == "" {
		if l := appSettings.LogLevelSlog(); l != logLevelDefault {
			slog.Info("Setting log level", "level", l)
			slog.SetLogLoggerLevel(l)
		}
	}

	// setup logfile for desktop
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		log.Fatal(err)
	}
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
		client, err := eveauth.NewClient(eveauth.Config{
			ClientID:   authClientID,
			IsDemoMode: true,
			Port:       authPort,
		})
		if err != nil {
			log.Fatal(err)
		}
		client.Authorize(context.Background(), []string{})
		return
	}

	if isDesktop {
		// ensure single instance
		mu, err := ensureSingleInstance()
		if errors.Is(err, mutex.ErrTimeout) {
			err := remoteservice.ShowPrimaryInstance(remotePort)
			if err != nil {
				log.Fatal(err)
			}
			slog.Info("Terminating secondary instance")
			os.Exit(0)
		}
		if err != nil {
			log.Fatal(err)
		}
		defer mu.Release()
		slog.Info("Identified as primary instance")
	}

	// crashfile
	if err := setupCrashFile(crashFilePath); err != nil {
		log.Fatalf("Failed to setup crash report file: %s", err)
	}

	if *deleteDataNoConfirmFlag {
		deleteapp.RemoveSettings(fyneApp)
		err := deleteapp.RemoveFolders(context.Background(), dataDir, nil)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	// start uninstall app if requested
	if isDesktop && *deleteDataFlag {
		u := deleteapp.NewUI(fyneApp)
		u.DataDir = dataDir
		u.ShowAndRun()
		return
	}

	// start pprof web server
	if *pprofFlag {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// init database
	dsn := "file:///" + filepath.ToSlash(dbPath)
	dbRW, dbRO, err := storage.InitDB(dsn)
	if err != nil {
		slog.Error("Failed to initialize database", "dsn", dsn, "error", err)
		os.Exit(1)
	}
	defer dbRW.Close()
	defer dbRO.Close()
	st := storage.New(dbRW, dbRO)

	// Initialize persistent cache
	pc := pcache.New(st, cacheCleanUpTimeout)
	defer pc.Close()

	if *clearCacheFlag {
		pc.Clear()
		slog.Info("cache cleared")
	}

	// HTTP client for ESI with automatic retries, HTTP caching, rate limit support,
	// error limit support, blocking during daily downtime period and response logging.
	rhc1 := retryablehttp.NewClient()
	rhc1.RetryWaitMax = 30 * time.Second // overruled by retry-after and error-reset header
	rhc1.RetryMax = 3
	rhc1.CheckRetry = customCheckRetry // also retry on 420s
	rhc1.Backoff = customBackoff       // also retry on 420s
	rhc1.HTTPClient.Transport = &httpcache.Transport{
		Cache:               newHTTPCacheAdapter(pc, "esicache-", 24*time.Hour),
		MarkCachedResponses: true,
		Transport: &xgoesi.RateLimiter{
			Transport: &xgoesi.DowntimeBlocker{},
		},
	}
	rhc1.Logger = slog.Default()
	rhc1.ResponseLogHook = logResponse
	userAgent := fmt.Sprintf("%s/%s (%s; +%s)", appName, fyneApp.Metadata().Version, userAgentEmail, sourceURL)
	esiClient := goesi.NewAPIClient(rhc1.StandardClient(), userAgent)
	slog.Info("user agent", "str", userAgent)

	// HTTP client for SSO and EVE image server with HTTP caching and response logging.
	rhc2 := retryablehttp.NewClient()
	rhc2.RetryWaitMax = 30 * time.Second
	rhc2.RetryMax = 4
	rhc2.HTTPClient.Transport = &httpcache.Transport{
		Cache:               newHTTPCacheAdapter(pc, "httpcache-", 24*time.Hour),
		MarkCachedResponses: true,
	}
	rhc2.Logger = slog.Default()
	rhc2.ResponseLogHook = logResponse

	// Init StatusCache service
	memCache := memcache.New()
	defer memCache.Close()
	scs := statuscacheservice.New(memCache, st)
	if err := scs.InitCache(context.Background()); err != nil {
		slog.Error("Failed to init cache", "error", err)
		os.Exit(1)
	}
	// Init EveUniverse service
	eus := eveuniverseservice.New(eveuniverseservice.Params{
		ConcurrencyLimit:   concurrentLimit,
		ESIClient:          esiClient,
		StatusCacheService: scs,
		Storage:            st,
	})

	// Init Character service
	authClient, err := eveauth.NewClient(eveauth.Config{
		ClientID:   authClientID,
		HTTPClient: rhc2.StandardClient(),
		Port:       authPort,
		OpenURL: func(u string) error {
			u2, err := url.ParseRequestURI(u)
			if err != nil {
				return err
			}
			if err := fyneApp.OpenURL(u2); err != nil {
				return err
			}
			return nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	cs := characterservice.New(characterservice.Params{
		Cache:                  newServiceCacheAdapter(pc, "characterservice-"),
		ConcurrencyLimit:       concurrentLimit,
		ESIClient:              esiClient,
		EveNotificationService: evenotification.New(eus),
		EveUniverseService:     eus,
		HTTPClient:             rhc1.StandardClient(),
		AuthClient:             authClient,
		StatusCacheService:     scs,
		Storage:                st,
	})

	// Init Corporation service
	rs := corporationservice.New(corporationservice.Params{
		Cache:              newServiceCacheAdapter(pc, "corporationservice-"),
		CharacterService:   cs,
		ConcurrencyLimit:   concurrentLimit,
		EsiClient:          esiClient,
		EveUniverseService: eus,
		HTTPClient:         rhc1.StandardClient(),
		StatusCacheService: scs,
		Storage:            st,
	})

	// Init UI
	os.Setenv("FYNE_SCALE", fmt.Sprint(appSettings.FyneScale()))
	os.Setenv("FYNE_DISABLE_DPI_DETECTION", fmt.Sprint(appSettings.DisableDPIDetection()))
	key := os.Getenv("JANICE_API_KEY")
	if key == "" {
		key = fyneApp.Metadata().Custom["janiceAPIKey"]
	}
	slog.Info("Janice API key", "value", obfuscate(key, 4, "X"))
	bu := ui.NewBaseUI(ui.BaseUIParams{
		App:              fyneApp,
		CharacterService: cs,
		ClearCacheFunc: func() {
			pc.Clear()
			memCache.Clear()

		},
		ConcurrencyLimit:   concurrentLimit,
		CorporationService: rs,
		DataPaths:          dataPaths,
		ESIStatusService:   esistatusservice.New(esiClient),
		EveImageService:    eveimageservice.New(pc, rhc2.StandardClient(), *offlineFlag),
		EveUniverseService: eus,
		IsMobile:           *mobileFlag || fyne.CurrentDevice().IsMobile(),
		IsFakeMobile:       *mobileFlag,
		IsOffline:          *offlineFlag,
		IsUpdateDisabled:   *disableUpdatesFlag,
		JaniceService:      janiceservice.New(rhc1.StandardClient(), key),
		MemCache:           memCache,
		StatusCacheService: scs,
	})
	if isDesktop {
		u := ui.NewDesktopUI(bu)
		stop, err := remoteservice.Start(remotePort, func() {
			fyne.Do(func() {
				u.MainWindow().Show()
			})
		})
		if err != nil {
			log.Fatal(err)
		}
		defer stop()
		u.ShowAndRun()
	} else {
		u := ui.NewMobileUI(bu)
		u.ShowAndRun()
	}
}

// obfuscate returns a new string of the same length as s with all characters replaced
// with a placeholder, except for the last n characters.
func obfuscate(s string, n int, placeholder string) string {
	if n > len(s) || n < 0 {
		return strings.Repeat(placeholder, len(s))
	}
	return strings.Repeat(placeholder, len(s)-n) + s[len(s)-n:]
}

// realtime represents the current time.
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
		return nil, fmt.Errorf("another instance running: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("acquire mutex: %w", err)
	}
	slog.Info("No other instances running")
	return mu, nil
}

// setupCrashFile create a dedicated file for storing crash reports and returns it's path.
func setupCrashFile(path string) error {
	crashFile, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	if err := debug.SetCrashOutput(crashFile, debug.CrashOptions{}); err != nil {
		return err
	}
	crashFile.Close()
	return nil
}

// customCheckRetry is a custom retry policy for a retryablehttp client
// that adds retry for 420s.
func customCheckRetry(ctx context.Context, resp *http.Response, err error) (bool, error) {
	shouldRetry, checkErr := retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	if checkErr != nil {
		return false, checkErr
	}
	if shouldRetry {
		return true, nil
	}
	if resp != nil && resp.StatusCode == xgoesi.StatusTooManyErrors {
		return true, nil // Retry on 420
	}
	return false, nil
}

// customBackoff is a custom backoff policy for a retryablehttp client
// that adds backoff for 420s.
func customBackoff(min time.Duration, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == xgoesi.StatusTooManyErrors {
		if sleep, ok := xgoesi.ParseErrorLimitResetHeader(resp); ok {
			return sleep
		}
		return xgoesi.ErrorLimitResetFallback
	}
	return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
}
