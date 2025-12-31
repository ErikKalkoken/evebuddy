package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"html/template"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

//go:embed target.go.tmpl
var tmplGo string

//go:embed target.md.tmpl
var tmplMD string

//go:embed openapi.json
var specFile []byte

var (
	packageFlag = flag.String("p", "main", "package name")
	formatFlat  = flag.String("f", "go", "format of generated output")
	output      = flag.String("out", "", "writes to given file when specified")
)

type rateLimitGroup struct {
	Name         string
	MaxTokens    int
	WindowSize   int     // seconds
	AverageRate  float64 // requests per second
	AverageDelay float64 // seconds
}

func main() {
	flag.Parse()
	var data map[string]any
	err := json.Unmarshal(specFile, &data)
	if err != nil {
		log.Fatal(err)
	}

	x1 := data["components"].(map[string]any)
	x2 := x1["parameters"].(map[string]any)
	x3 := x2["CompatibilityDate"].(map[string]any)
	x4 := x3["schema"].(map[string]any)
	x5 := x4["enum"].([]any)
	compatibilityDate := x5[0].(string)

	groups := make(map[string]rateLimitGroup)
	operations := make(map[string]string)
	tags := make(map[string]map[string]string)

	for _, v := range data["paths"].(map[string]any) {
		for _, v := range v.(map[string]any) {
			route := v.(map[string]any)
			operationID := route["operationId"].(string)
			tag := route["tags"].([]any)[0].(string)
			_, ok := tags[tag]
			if !ok {
				tags[tag] = make(map[string]string)
			}
			x2, ok := route["x-rate-limit"]
			if !ok {
				tags[tag][operationID] = ""
				operations[operationID] = ""
				continue
			}
			rateLimit := x2.(map[string]any)
			var rl rateLimitGroup
			rl.Name = rateLimit["group"].(string)
			maxTokens := rateLimit["max-tokens"].(float64)
			rl.MaxTokens = int(maxTokens)
			x3 := rateLimit["window-size"].(string)
			d, err := time.ParseDuration(x3)
			if err != nil {
				log.Fatal(err)
			}
			rl.WindowSize = int(d.Seconds())
			rl.AverageDelay = d.Seconds() / (maxTokens / 2)
			rl.AverageRate = 1 / rl.AverageDelay
			tags[tag][operationID] = rl.Name
			operations[operationID] = rl.Name
			_, found := groups[rl.Name]
			if found {
				continue
			}
			groups[rl.Name] = rl
		}
	}

	var out io.Writer
	if *output == "" {
		out = os.Stdout
	} else {
		f, err := os.Create(*output)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		out = f
	}

	switch *formatFlat {
	case "md":
		tmpl, err := template.New("").Funcs(template.FuncMap{
			"toAnchor": toAnchor,
		}).Parse(tmplMD)
		if err != nil {
			log.Fatal(err)
		}
		err = tmpl.Execute(out, map[string]any{
			"Today":             time.Now().UTC().Format(time.DateOnly),
			"CompatibilityDate": compatibilityDate,
			"Groups":            groups,
			"Operations":        operations,
			"Tags":              tags,
		})
		if err != nil {
			log.Fatal(err)
		}
	default:
		tmpl, err := template.New("").Parse(tmplGo)
		if err != nil {
			log.Fatal(err)
		}
		err = tmpl.Execute(out, map[string]any{
			"CompatibilityDate": compatibilityDate,
			"Groups":            groups,
			"Operations":        operations,
			"Package":           *packageFlag,
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func toAnchor(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
