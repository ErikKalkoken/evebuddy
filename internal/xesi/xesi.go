// Package xesi contains extensions to the esi package.
package xesi

import (
	"net/http"
	"slices"
	"strconv"

	"golang.org/x/sync/errgroup"
)

// FetchPages fetches and returns the combined list of items
// from all pages of an ESI endpoint that supports paging with X-Pages.
// Subsequent pages are fetched concurrently.
func FetchPages[T any](concurrencyLimit int, fetch func(page int) ([]T, *http.Response, error)) ([]T, error) {
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
	g.SetLimit(concurrencyLimit)
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

// FetchPagesWithExit fetches and returns the combined list of items
// from all pages of an ESI endpoint that supports paging with X-Pages.
// Will stop fetching subsequent pages when a page returns an item for which the found function returns true.
// If found is nil it will be ignored.
func FetchPagesWithExit[T any](fetch func(page int) ([]T, *http.Response, error), found func(x T) bool) ([]T, error) {
	exit := func(s []T) bool {
		if found == nil {
			return false
		}
		return slices.ContainsFunc(s, found)
	}
	items, r, err := fetch(1)
	if err != nil {
		return nil, err
	}
	pages, err := extractPageCount(r)
	if err != nil {
		return nil, err
	}
	if pages < 2 || exit(items) {
		return items, nil
	}
	for p := 2; p <= pages; p++ {
		it, _, err := fetch(p)
		if err != nil {
			return nil, err
		}
		items = slices.Concat(items, it)
		if exit(it) {
			break
		}
	}
	return items, nil
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
