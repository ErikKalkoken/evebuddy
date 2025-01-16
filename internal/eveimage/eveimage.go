// Package eveimage provides cached access to images from the Eve Online image server.
package eveimage

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"golang.org/x/sync/singleflight"
)

// cache timeouts per image category
const (
	timeoutAlliance    = time.Hour * 24 * 3
	timeoutCharacter   = time.Hour * 24 * 1
	timeoutCorporation = time.Hour * 24 * 1
	timeoutDefault     = 0
)

var (
	ErrHttpError   = errors.New("http error")
	ErrNoImage     = errors.New("no image from API")
	ErrInvalidSize = errors.New("invalid size")
)

// Defines a cache service
type CacheService interface {
	Clear()
	Get(string) ([]byte, bool)
	Set(string, []byte, time.Duration)
}

// EveImageService provides cached access to images on the Eve Online image server.
type EveImageService struct {
	cache      CacheService
	httpClient *http.Client
	isOffline  bool
	sfg        *singleflight.Group
}

// New returns a new EveImageService object.
//
// When no httpClient (nil) is provided it will use the default client.
// When isOffline is set to true, it will return a dummy image
// instead of trying to fetch images from the image server, which are not already cached.
func New(cache CacheService, httpClient *http.Client, isOffline bool) *EveImageService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	m := &EveImageService{
		cache:      cache,
		httpClient: httpClient,
		isOffline:  isOffline,
		sfg:        new(singleflight.Group),
	}
	return m
}

// AllianceLogo returns the logo for an alliance.
func (m *EveImageService) AllianceLogo(id int32, size int) (fyne.Resource, error) {
	url, err := AllianceLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url, timeoutAlliance)
}

// CharacterPortrait returns the portrait for a character.
func (m *EveImageService) CharacterPortrait(id int32, size int) (fyne.Resource, error) {
	url, err := CharacterPortraitURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url, timeoutCharacter)
}

// CorporationLogo returns the logo for a corporation.
func (m *EveImageService) CorporationLogo(id int32, size int) (fyne.Resource, error) {
	url, err := CorporationLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url, timeoutCorporation)
}

// FactionLogo returns the logo for a faction.
func (m *EveImageService) FactionLogo(id int32, size int) (fyne.Resource, error) {
	url, err := FactionLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url, timeoutDefault)
}

// InventoryTypeRender returns the render for a type. Note that not ever type has a render.
func (m *EveImageService) InventoryTypeRender(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeRenderURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url, timeoutDefault)
}

// InventoryTypeIcon returns the icon for a type.
func (m *EveImageService) InventoryTypeIcon(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeIconURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url, timeoutDefault)
}

// InventoryTypeBPO returns the icon for a BPO type.
func (m *EveImageService) InventoryTypeBPO(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeBPOURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url, timeoutDefault)
}

// InventoryTypeBPC returns the icon for a BPC type.
func (m *EveImageService) InventoryTypeBPC(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeBPCURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return m.image(url, timeoutDefault)
}

// InventoryTypeSKIN returns the icon for a SKIN type.
func (m *EveImageService) InventoryTypeSKIN(id int32, size int) (fyne.Resource, error) {
	if size != 64 {
		return nil, ErrInvalidSize
	}
	return resourceSkinicon64pxPng, nil
}

// image returns an Eve image as fyne resource.
// It returns it from cache or - if not found - will try to fetch it from the Internet.
func (m *EveImageService) image(url string, timeout time.Duration) (fyne.Resource, error) {
	key := "eveimage-" + makeMD5Hash(url)
	var dat []byte
	var found bool
	dat, found = m.cache.Get(key)
	if !found {
		if m.isOffline {
			return resourceBrokenimageSvg, nil
		}
		x, err, _ := m.sfg.Do(key, func() (any, error) {
			byt, err := loadDataFromURL(url, m.httpClient)
			if err != nil {
				return nil, err
			}
			m.cache.Set(key, byt, timeout)
			return byt, nil
		})
		if err != nil {
			return nil, err
		}
		dat = x.([]byte)
	}
	r := fyne.NewStaticResource(key, dat)
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
func (m *EveImageService) ClearCache() error {
	m.cache.Clear()
	return nil
}
