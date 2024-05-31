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

func fetchFromESIWithPaging[T any](fetch func(int) ([]T, *http.Response, error)) ([]T, error) {
	results := make(map[int][]T)
	page := 1
	for {
		result, r, err := fetch(page)
		if err != nil {
			return nil, err
		}
		results[page] = result
		pages := 1
		x := r.Header.Get("X-Pages")
		if x != "" {
			pages, err = strconv.Atoi(x)
			if err != nil {
				return nil, err
			}
		}
		if page == pages {
			break
		}
		page++
	}
	assetsAll := make([]T, 0)
	for _, result := range results {
		assetsAll = slices.Concat(assetsAll, result)
	}
	return assetsAll, nil
}
