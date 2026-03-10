// Package eveimageservice contains the EVE image service.
package eveimageservice

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"golang.org/x/sync/singleflight"

	"github.com/ErikKalkoken/evebuddy/internal/xsingleflight"
)

// cache timeouts per image category
const (
	timeoutAlliance    = time.Hour * 24 * 7
	timeoutCharacter   = time.Hour * 24 * 1
	timeoutCorporation = time.Hour * 24 * 3
	timeoutNeverExpire = 0
)

var (
	ErrHTTPError = errors.New("http error")
	ErrNoImage   = errors.New("no image from API")
	ErrInvalid   = errors.New("invalid")
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

// EVEImageService represents a service which provides access to images on the Eve Online image server.
// Images are cached.
type EVEImageService struct {
	cache      CacheService
	httpClient *http.Client
	isOffline  bool
	sfg        singleflight.Group
}

// New returns a new EveImageService.
//
// When no httpClient (nil) is provided it will use the default client.
// When isOffline is set to true, it will return a dummy image
// instead of trying to fetch images from the image server, which are not already cached.
func New(cache CacheService, httpClient *http.Client, isOffline bool) *EVEImageService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	s := &EVEImageService{
		cache:      cache,
		httpClient: httpClient,
		isOffline:  isOffline,
	}
	return s
}

// AllianceLogo returns the logo for an alliance.
func (s *EVEImageService) AllianceLogo(id int64, size int) (fyne.Resource, error) {
	url, err := AllianceLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			err = ErrInvalid
		}
		return nil, err
	}
	return s.image(url, timeoutAlliance)
}

// AllianceLogoAsync loads an alliance logo asynchronously and calls setter with the result.
func (s *EVEImageService) AllianceLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	s.loadImageAsync(loadImageAsyncParams{
		id:      id,
		makeURL: AllianceLogoURL,
		setter:  setter,
		size:    size,
		timeout: timeoutAlliance,
	})
}

// CharacterPortrait returns the portrait for a character.
func (s *EVEImageService) CharacterPortrait(id int64, size int) (fyne.Resource, error) {
	url, err := CharacterPortraitURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			err = ErrInvalid
		}
		return nil, err
	}
	return s.image(url, timeoutCharacter)
}

// CharacterPortraitAsync loads a character portrait asynchronously and calls setter with the result.
func (s *EVEImageService) CharacterPortraitAsync(id int64, size int, setter func(r fyne.Resource)) {
	s.loadImageAsync(loadImageAsyncParams{
		id:      id,
		makeURL: CharacterPortraitURL,
		setter:  setter,
		size:    size,
		timeout: timeoutCharacter,
	})
}

// CorporationLogo returns the logo for a corporation.
func (s *EVEImageService) CorporationLogo(id int64, size int) (fyne.Resource, error) {
	url, err := CorporationLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			err = ErrInvalid
		}
		return nil, err
	}
	return s.image(url, timeoutCorporation)
}

// CorporationLogoAsync loads a character portrait asynchronously and calls setter with the result.
func (s *EVEImageService) CorporationLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	s.loadImageAsync(loadImageAsyncParams{
		id:      id,
		makeURL: CorporationLogoURL,
		setter:  setter,
		size:    size,
		timeout: timeoutCorporation,
	})
}

// FactionLogo returns the logo for a faction.
func (s *EVEImageService) FactionLogo(id int64, size int) (fyne.Resource, error) {
	url, err := FactionLogoURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			err = ErrInvalid
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// FactionLogoAsync loads a faction logo asynchronously and calls setter with the result.
func (s *EVEImageService) FactionLogoAsync(id int64, size int, setter func(r fyne.Resource)) {
	s.loadImageAsync(loadImageAsyncParams{
		id:      id,
		makeURL: FactionLogoURL,
		setter:  setter,
		size:    size,
		timeout: timeoutNeverExpire,
	})
}

// InventoryTypeRender returns the render for a type. Note that not ever type has a render.
func (s *EVEImageService) InventoryTypeRender(id int64, size int) (fyne.Resource, error) {
	url, err := InventoryTypeRenderURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			err = ErrInvalid
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeRenderAsync loads a render for a type and calls setter with the result.
func (s *EVEImageService) InventoryTypeRenderAsync(id int64, size int, setter func(r fyne.Resource)) {
	s.loadImageAsync(loadImageAsyncParams{
		id:      id,
		makeURL: InventoryTypeRenderURL,
		setter:  setter,
		size:    size,
		timeout: timeoutNeverExpire,
	})
}

// InventoryTypeIcon returns the icon for a type.
func (s *EVEImageService) InventoryTypeIcon(id int64, size int) (fyne.Resource, error) {
	url, err := InventoryTypeIconURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			err = ErrInvalid
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeIconAsync loads a render for a type and calls setter with the result.
func (s *EVEImageService) InventoryTypeIconAsync(id int64, size int, setter func(r fyne.Resource)) {
	s.loadImageAsync(loadImageAsyncParams{
		id:      id,
		makeURL: InventoryTypeIconURL,
		setter:  setter,
		size:    size,
		timeout: timeoutNeverExpire,
	})
}

// InventoryTypeBPO returns the icon for a BPO type.
func (s *EVEImageService) InventoryTypeBPO(id int64, size int) (fyne.Resource, error) {
	url, err := InventoryTypeBPOURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			err = ErrInvalid
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeBPOAsync loads the icon for a BPO type and calls setter with the result.
func (s *EVEImageService) InventoryTypeBPOAsync(id int64, size int, setter func(r fyne.Resource)) {
	s.loadImageAsync(loadImageAsyncParams{
		id:      id,
		makeURL: InventoryTypeBPOURL,
		setter:  setter,
		size:    size,
		timeout: timeoutNeverExpire,
	})
}

// InventoryTypeBPC returns the icon for a BPC type.
func (s *EVEImageService) InventoryTypeBPC(id int64, size int) (fyne.Resource, error) {
	url, err := InventoryTypeBPCURL(id, size)
	if err != nil {
		if errors.Is(err, ErrInvalid) {
			err = ErrInvalid
		}
		return nil, err
	}
	return s.image(url, timeoutNeverExpire)
}

// InventoryTypeBPCAsync loads the icon for a BPC type and calls setter with the result.
func (s *EVEImageService) InventoryTypeBPCAsync(id int64, size int, setter func(r fyne.Resource)) {
	s.loadImageAsync(loadImageAsyncParams{
		id:      id,
		makeURL: InventoryTypeBPCURL,
		setter:  setter,
		size:    size,
		timeout: timeoutNeverExpire,
	})
}

// InventoryTypeSKIN returns the icon for a SKIN type.
func (s *EVEImageService) InventoryTypeSKIN(_ int64, size int) (fyne.Resource, error) {
	if size != 64 {
		return nil, ErrInvalid
	}
	return resourceSkinicon64pxPng, nil
}

// InventoryTypeSKINAsync loads the icon for a SKIN type and calls setter with the result.
func (s *EVEImageService) InventoryTypeSKINAsync(_ int64, size int, setter func(r fyne.Resource)) {
	if size != 64 {
		slog.Error("eveimageservice: url", "error", ErrInvalid)
		setter(resourceBrokenimage64Png)
		return
	}
	setter(resourceSkinicon64pxPng)
}

type loadImageAsyncParams struct {
	id      int64
	makeURL func(id int64, size int) (string, error)
	setter  func(r fyne.Resource)
	size    int
	timeout time.Duration
}

func (s *EVEImageService) loadImageAsync(arg loadImageAsyncParams) {
	url, err := arg.makeURL(arg.id, arg.size)
	if err != nil {
		slog.Error("eveimageservice: url", "error", err)
		arg.setter(resourceBrokenimage64Png)
		return
	}
	key := makeKey(url)
	dat, found := s.cache.Get(key)
	if found {
		arg.setter(fyne.NewStaticResource(key, dat))
		return
	}
	if s.isOffline {
		arg.setter(resourceBrokenimage64Png)
		return
	}
	arg.setter(resourceBlank32Png)
	go func() {
		dat, err, _ := xsingleflight.Do(&s.sfg, key, func() ([]byte, error) {
			byt, err := loadDataFromURL(url, s.httpClient)
			if err != nil {
				return nil, err
			}
			s.cache.Set(key, byt, arg.timeout)
			return byt, nil
		})
		if err != nil {
			slog.Error("eveimageservice: request", "error", err)
			fyne.Do(func() {
				arg.setter(resourceBrokenimage64Png)
			})
			return
		}
		fyne.Do(func() {
			arg.setter(fyne.NewStaticResource(key, dat))
		})
	}()
}

// image returns an Eve image as fyne resource.
// It returns it from cache or - if not found - will try to fetch it from the Internet.
func (s *EVEImageService) image(url string, timeout time.Duration) (fyne.Resource, error) {
	key := makeKey(url)
	dat, found := s.cache.Get(key)
	if !found {
		if s.isOffline {
			return resourceQuestionmark32Png, nil
		}
		v, err, _ := xsingleflight.Do(&s.sfg, key, func() ([]byte, error) {
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
		dat = v
	}
	r := fyne.NewStaticResource(key, dat)
	return r, nil
}

func makeKey(url string) string {
	key := "eveimage-" + makeMD5Hash(url)
	return key
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
