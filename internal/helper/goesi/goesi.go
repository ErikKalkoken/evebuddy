// package goesi contains helpers used in conjunction with the goesi package
package goesi

import (
	"net/http"
	"slices"
	"strconv"

	"golang.org/x/sync/errgroup"
)

// FetchFromESIWithPaging returns the combined list of items from all pages of an ESI endpoint.
// This only works for ESI endpoints which support the X-Pages pattern and return a list.
func FetchFromESIWithPaging[T any](fetch func(int) ([]T, *http.Response, error)) ([]T, error) {
	results := make(map[int][]T)
	result, r, err := fetch(1)
	if err != nil {
		return nil, err
	}
	results[1] = result
	pages, err := extractPageCount(r)
	if err != nil {
		return nil, err
	}
	if pages > 1 {
		g := new(errgroup.Group)
		for p := 2; p <= pages; p++ {
			p := p
			g.Go(func() error {
				result, _, err := fetch(p)
				if err != nil {
					return err
				}
				results[p] = result
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
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
