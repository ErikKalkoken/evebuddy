package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"os"
	"time"
)

//go:embed target.go.template
var tmpl string

//go:embed openapi.json
var specFile []byte

var packageFlag = flag.String("p", "main", "package name")

type rateLimitGroup struct {
	Name       string
	MaxTokens  int
	WindowSize int // seconds
}

func main() {
	flag.Parse()
	var data map[string]any
	err := json.Unmarshal(specFile, &data)
	if err != nil {
		log.Fatal(err)
	}
	groups := make(map[string]rateLimitGroup)
	operations := make(map[string]string)

	x1 := data["components"].(map[string]any)
	x2 := x1["parameters"].(map[string]any)
	x3 := x2["CompatibilityDate"].(map[string]any)
	x4 := x3["schema"].(map[string]any)
	x5 := x4["enum"].([]any)
	compatibilityDate := x5[0].(string)

	paths := data["paths"].(map[string]any)
	for _, v := range paths {
		method := v.(map[string]any)
		for _, v := range method {
			route := v.(map[string]any)
			operationID := route["operationId"].(string)
			x2, ok := route["x-rate-limit"]
			if !ok {
				operations[operationID] = ""
				continue
			}
			rateLimit := x2.(map[string]any)
			var rl rateLimitGroup
			rl.Name = rateLimit["group"].(string)
			rl.MaxTokens = int(rateLimit["max-tokens"].(float64))
			x3 := rateLimit["window-size"].(string)
			d, err := time.ParseDuration(x3)
			if err != nil {
				log.Fatal(err)
			}
			rl.WindowSize = int(d.Seconds())
			operations[operationID] = rl.Name
			_, found := groups[rl.Name]
			if found {
				continue
			}
			groups[rl.Name] = rl
		}
	}

	tmpl, err := template.New("").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.Execute(os.Stdout, map[string]any{
		"Package":           *packageFlag,
		"Operations":        operations,
		"Groups":            groups,
		"CompatibilityDate": compatibilityDate,
	})
	if err != nil {
		log.Fatal(err)
	}
}
