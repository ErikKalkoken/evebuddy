package appdirs

import (
	"os"
	"path/filepath"

	xappdirs "github.com/chasinglogic/appdirs"
)

const (
	appName       = "evebuddy"
	logFolderName = "log"
)

// AppDirs represents the app's local directories for storing logs etc.
type AppDirs struct {
	Data     string
	Log      string
	Settings string
}

func New() (AppDirs, error) {
	ad := xappdirs.New(appName)
	x := AppDirs{
		Data: ad.UserData(),
		Log:  filepath.Join(ad.UserData(), logFolderName),
	}
	if err := os.MkdirAll(x.Log, os.ModePerm); err != nil {
		return x, err
	}
	if err := os.MkdirAll(x.Data, os.ModePerm); err != nil {
		return x, err
	}
	return x, nil
}

func (ad *AppDirs) SetSettings(p string) {
	ad.Settings = p
}

func (ad *AppDirs) Folders() []string {
	return []string{ad.Log, ad.Data, ad.Settings}
}
