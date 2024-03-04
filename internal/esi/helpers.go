package esi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func UnmarshalResponse[T any](resp *http.Response) (T, error) {
	var objs T
	if resp.Body != nil {
		defer resp.Body.Close()
	} else {
		return objs, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return objs, err
	}
	if err := json.Unmarshal(body, &objs); err != nil {
		log.Fatal(err)
	}
	return objs, nil
}
