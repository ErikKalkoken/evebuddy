package service

import (
	"net/http"
	"slices"
	"strconv"
)

func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

// TODO: Fetch pages in parallel

// fetchFromESIWithPaging returns the combined list of items from all pages of an ESI endpoint.
// This only works for ESI endpoints which support the X-Pages pattern and return a list.
func fetchFromESIWithPaging[T any](fetch func(int) ([]T, *http.Response, error)) ([]T, error) {
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
		for p := 2; p <= pages; p++ {
			result, _, err := fetch(p)
			if err != nil {
				return nil, err
			}
			results[p] = result
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
