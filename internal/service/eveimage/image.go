package eveimage

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"fyne.io/fyne/v2"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/images"
)

var (
	ErrHttpError   = errors.New("http error")
	ErrNoImage     = errors.New("no image from API")
	ErrInvalidSize = errors.New("invalid size")
)

// EveImageService provides cached access to images from the Eve Online image server.
type EveImageService struct {
	httpClient *http.Client
	// cacheDir is where the image files are stored for caching
	cacheDir string
	sfg      *singleflight.Group
}

// New returns a new Images object. path is the location of the file cache.
// When no path is given (empty string) it will create a temporary directory instead.
// When not provides a httpClient (nil) it will use the default client.
func New(cacheDir string, httpClient *http.Client) *EveImageService {
	if cacheDir == "" {
		p, err := os.MkdirTemp("", "eveimage")
		if err != nil {
			log.Fatal(err)
		}
		cacheDir = p
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	m := &EveImageService{
		httpClient: httpClient,
		cacheDir:   cacheDir,
		sfg:        new(singleflight.Group),
	}
	return m
}

// AllianceLogo returns the logo for an alliance.
func (m *EveImageService) AllianceLogo(id int32, size int) (fyne.Resource, error) {
	url, err := images.AllianceLogoURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// CharacterPortrait returns the portrait for a character.
func (m *EveImageService) CharacterPortrait(id int32, size int) (fyne.Resource, error) {
	url, err := images.CharacterPortraitURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// CorporationLogo returns the logo for a corporation.
func (m *EveImageService) CorporationLogo(id int32, size int) (fyne.Resource, error) {
	url, err := images.CorporationLogoURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// FactionLogo returns the logo for a faction.
func (m *EveImageService) FactionLogo(id int32, size int) (fyne.Resource, error) {
	url, err := images.FactionLogoURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// InventoryTypeRender returns the render for a type. Note that not ever type has a render.
func (m *EveImageService) InventoryTypeRender(id int32, size int) (fyne.Resource, error) {
	url, err := images.InventoryTypeRenderURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// InventoryTypeIcon returns the icon for a type.
func (m *EveImageService) InventoryTypeIcon(id int32, size int) (fyne.Resource, error) {
	url, err := images.InventoryTypeIconURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// InventoryTypeBPO returns the icon for a BPO type.
func (m *EveImageService) InventoryTypeBPO(id int32, size int) (fyne.Resource, error) {
	url, err := images.InventoryTypeBPOURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

// InventoryTypeBPC returns the icon for a BPC type.
func (m *EveImageService) InventoryTypeBPC(id int32, size int) (fyne.Resource, error) {
	url, err := images.InventoryTypeBPCURL(id, size)
	if err != nil {
		if errors.Is(err, images.ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url)
}

func (m *EveImageService) image(url string) (fyne.Resource, error) {
	hash := makeMD5Hash(url)
	name := filepath.Join(m.cacheDir, hash+".tmp")
	dat, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		x, err, _ := m.sfg.Do(hash, func() (any, error) {
			dat, err = loadDataFromURL(url, m.httpClient)
			if err != nil {
				return nil, err
			}
			if err := os.WriteFile(name, dat, 0666); err != nil {
				return nil, err
			}
			return dat, nil
		})
		if err != nil {
			return nil, err
		}
		dat = x.([]byte)
	} else if err != nil {
		return nil, err
	}
	r := fyne.NewStaticResource(fmt.Sprintf("eve-image-%s", hash), dat)
	return r, nil
}

type HTTPError struct {
	StatusCode int
	Status     string
}

func (r HTTPError) Error() string {
	return fmt.Sprintf("HTTP error: %s", r.Status)
}

func loadDataFromURL(url string, client *http.Client) ([]byte, error) {
	r, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if r.StatusCode >= 400 {
		err := HTTPError{StatusCode: r.StatusCode, Status: r.Status}
		return nil, err
	}
	if r.Body == nil {
		return nil, fmt.Errorf("%s: %w", url, ErrNoImage)
	}
	dat, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func makeMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// ClearCache clears the images cache and returns the number of deleted entries.
func (m *EveImageService) ClearCache() (int, error) {
	files, err := os.ReadDir(m.cacheDir)
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		os.RemoveAll(path.Join(m.cacheDir, f.Name()))
	}
	return len(files), nil
}

// Size returns the total size of all image files in by bytes.
func (m *EveImageService) Size() (int, error) {
	files, err := os.ReadDir(m.cacheDir)
	if err != nil {
		return 0, err
	}
	var s int64
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			return 0, err
		}
		s += info.Size()
	}
	return int(s), nil
}

// Count returns the number of all image files.
func (m *EveImageService) Count() (int, error) {
	files, err := os.ReadDir(m.cacheDir)
	if err != nil {
		return 0, err
	}
	return len(files), nil
}
