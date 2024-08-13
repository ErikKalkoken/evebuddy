// package goesi contains helpers used in conjunction with the goesi package
package character

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/antihax/goesi"
	"golang.org/x/sync/errgroup"
)

// contextWithESIToken returns a new context with the ESI access token included
// so it can be used to authenticate requests with the goesi library.
func contextWithESIToken(ctx context.Context, accessToken string) context.Context {
	ctx = context.WithValue(ctx, goesi.ContextAccessToken, accessToken)
	return ctx
}

// fetchFromESIWithPaging returns the combined list of items from all pages of an ESI endpoint.
// This only works for ESI endpoints which support the X-Pages pattern and return a list.
func fetchFromESIWithPaging[T any](fetch func(int) ([]T, *http.Response, error)) ([]T, error) {
	result, r, err := fetch(1)
	if err != nil {
		return nil, err
	}
	pages, err := extractPageCount(r)
	if err != nil {
		return nil, err
	}
	if pages < 2 {
		return result, nil
	}
	results := make([][]T, pages)
	results[0] = result
	g := new(errgroup.Group)
	for p := 2; p <= pages; p++ {
		p := p
		g.Go(func() error {
			result, _, err := fetch(p)
			if err != nil {
				return err
			}
			results[p-1] = result
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	combined := make([]T, 0)
	for _, result := range results {
		combined = slices.Concat(combined, result)
	}
	return combined, nil
}

func extractPageCount(r *http.Response) (int, error) {
	x := r.Header.Get("X-Pages")
	if x == "" {
		return 1, nil
	}
	pages, err := strconv.Atoi(x)
	if err != nil {
		return 0, err
	}
	return pages, nil
}

// FromLDAPTime converts an ldap time to golang time
func FromLDAPTime(ldap_dt int64) time.Time {
	return time.Unix((ldap_dt/10000000)-11644473600, 0).UTC()
}

// FromLDAPDuration converts an ldap duration to golang duration
func FromLDAPDuration(ldap_td int64) time.Duration {
	return time.Duration(ldap_td/10) * time.Microsecond
}
