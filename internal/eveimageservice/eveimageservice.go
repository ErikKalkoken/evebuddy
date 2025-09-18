// Package eveimageservice contains the EVE image service.
package eveimageservice

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

// type Category string

// const (
// 	Alliance Category = "alliance"
// )

// cache timeouts per image category
const (
	timeoutAlliance    = time.Hour * 24 * 7
	timeoutCharacter   = time.Hour * 24 * 1
	timeoutCorporation = time.Hour * 24 * 3
	timeoutNeverExpire = 0
)

var (
	ErrHTTPError   = errors.New("http error")
	ErrNoImage     = errors.New("no image from API")
	ErrInvalidSize = errors.New("invalid size")
)

// CacheService defines a cache service
type CacheService interface {
	Get(string) ([]byte, bool)
	Set(string, []byte, time.Duration)
}

// HTTPError represents a HTTP response with status code >= 400.
type HTTPError struct {
	StatusCode int
	Status     string
}

func (r HTTPError) Error() string {
	return fmt.Sprintf("HTTP error: %s", r.Status)
}

// EveImageService represents a service which provides access to images on the Eve Online image server.
// Images are cached.
type EveImageService struct {
	cache      CacheService
	httpClient *http.Client
	isOffline  bool
	sfg        *singleflight.Group
}

// New returns a new EveImageService.
//
// When no httpClient (nil) is provided it will use the default client.
// When isOffline is set to true, it will return a dummy image
// instead of trying to fetch images from the image server, which are not already cached.
func New(cache CacheService, httpClient *http.Client, isOffline bool) *EveImageService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	s := &EveImageService{
		cache:      cache,
		httpClient: httpClient,
		isOffline:  isOffline,
		sfg:        new(singleflight.Group),
	}
	return s
}

// AllianceLogo returns the logo for an alliance.
func (s *EveImageService) AllianceLogo(id int32, size int) (fyne.Resource, error) {
	url, err := AllianceLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return s.image(url, timeoutAlliance)
}

// CharacterPortrait returns the portrait for a character.
func (s *EveImageService) CharacterPortrait(id int32, size int) (fyne.Resource, error) {
	url, err := CharacterPortraitURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return s.image(url, timeoutCharacter)
}

// CorporationLogo returns the logo for a corporation.
func (s *EveImageService) CorporationLogo(id int32, size int) (fyne.Resource, error) {
	url, err := CorporationLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return s.image(url, timeoutCorporation)
}

// FactionLogo returns the logo for a faction.
func (s *EveImageService) FactionLogo(id int32, size int) (fyne.Resource, error) {
	url, err := FactionLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeRender returns the render for a type. Note that not ever type has a render.
func (s *EveImageService) InventoryTypeRender(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeRenderURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeIcon returns the icon for a type.
func (s *EveImageService) InventoryTypeIcon(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeIconURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeBPO returns the icon for a BPO type.
func (s *EveImageService) InventoryTypeBPO(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeBPOURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeBPC returns the icon for a BPC type.
func (s *EveImageService) InventoryTypeBPC(id int32, size int) (fyne.Resource, error) {
	url, err := InventoryTypeBPCURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalidSize) {
			err = ErrInvalidSize
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeSKIN returns the icon for a SKIN type.
func (s *EveImageService) InventoryTypeSKIN(id int32, size int) (fyne.Resource, error) {
	if size != 64 {
		return nil, ErrInvalidSize
	}
	return resourceSkinicon64pxPng, nil
}

// image returns an Eve image as fyne resource.
// It returns it from cache or - if not found - will try to fetch it from the Internet.
func (s *EveImageService) image(url string, timeout time.Duration) (fyne.Resource, error) {
	key := "eveimage-" + makeMD5Hash(url)
	var dat []byte
	var found bool
	dat, found = s.cache.Get(key)
	if !found {
		if s.isOffline {
			return resourceQuestionmark32Png, nil
		}
		x, err, _ := s.sfg.Do(key, func() (any, error) {
			byt, err := loadDataFromURL(url, s.httpClient)
			if err != nil {
				return nil, err
			}
			s.cache.Set(key, byt, timeout)
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
