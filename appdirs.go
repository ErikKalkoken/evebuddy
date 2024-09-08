package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	xappdirs "github.com/chasinglogic/appdirs"
)

const (
	appName         = "evebuddy"
	logFileName     = "evebuddy.log"
	dbFileName      = "evebuddy.sqlite"
	cacheFolderName = "images"
)

// appDirs represents the app's local directories for storing logs etc.
type appDirs struct {
	cache    string
	data     string
	log      string
	settings string
}

func newAppDirs(fyneApp fyne.App) appDirs {
	ad := xappdirs.New(appName)
	x := appDirs{
		data:     ad.UserData(),
		cache:    ad.UserCache(),
		log:      ad.UserLog(),
		settings: fyneApp.Storage().RootURI().Path(),
	}
	return x
}

func (ad appDirs) deleteAll() error {
	for _, p := range []string{ad.log, ad.cache, ad.data, ad.settings} {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
		fmt.Printf("Deleted %s\n", p)
	}
	return nil
}

func (ad appDirs) initLogFile() (string, error) {
	if err := os.MkdirAll(ad.log, os.ModePerm); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", ad.log, logFileName), nil
}

func (ad appDirs) initDSN() (string, error) {
	if err := os.MkdirAll(ad.data, os.ModePerm); err != nil {
		return "", err
	}
	dsn := fmt.Sprintf("file:%s/%s", ad.data, dbFileName)
	return dsn, nil
}

func (ad appDirs) initImageCachePath() (string, error) {
	p := filepath.Join(ad.cache, cacheFolderName)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return "", err
	}
	return p, nil
}
