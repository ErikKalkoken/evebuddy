package appdirs

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	xappdirs "github.com/chasinglogic/appdirs"
)

const (
	appName         = "evebuddy"
	cacheFolderName = "images"
)

// AppDirs represents the app's local directories for storing logs etc.
type AppDirs struct {
	Cache    string
	Data     string
	Log      string
	Settings string
}

func New(fyneApp fyne.App) (AppDirs, error) {
	ad := xappdirs.New(appName)
	x := AppDirs{
		Data:     ad.UserData(),
		Cache:    ad.UserCache(),
		Log:      ad.UserLog(),
		Settings: fyneApp.Storage().RootURI().Path(),
	}
	if err := os.MkdirAll(x.Log, os.ModePerm); err != nil {
		return x, err
	}
	if err := os.MkdirAll(x.Data, os.ModePerm); err != nil {
		return x, err
	}
	p := filepath.Join(x.Cache, cacheFolderName)
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return x, err
	}
	return x, nil
}

func (ad AppDirs) DeleteAll() error {
	for _, p := range []string{ad.Log, ad.Cache, ad.Data, ad.Settings} {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}
