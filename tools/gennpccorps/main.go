// gennpccorps generates a go source file containing additional information
// about NPC corporations.
package main

import (
	"bytes"
	"cmp"
	_ "embed"
	"flag"
	"go/format"
	"log"
	"os"
	"slices"
	"text/template"

	"github.com/goccy/go-yaml"
)

//go:embed target.go.template
var tmpl string

//go:embed npcCorporations.yaml
var yamlData []byte

type corporation struct {
	FactionID int64 `yaml:"factionID"`
}

type row struct {
	CorporationID int64
	FactionID     int64
}

var (
	packageFlag = flag.String("p", "main", "package name")
	output      = flag.String("out", "", "writes to given file when specified")
)

func main() {
	flag.Parse()
	var data map[int64]corporation
	err := yaml.Unmarshal(yamlData, &data)
	if err != nil {
		log.Fatal(err)
	}

	var values []row
	for k, v := range data {
		if v.FactionID == 0 {
			continue
		}
		r := row{CorporationID: k, FactionID: v.FactionID}
		values = append(values, r)
	}
	slices.SortFunc(values, func(a, b row) int {
		return cmp.Compare(a.CorporationID, b.CorporationID)
	})
	tmpl, err := template.New("").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		"Package":  *packageFlag,
		"Values":   values,
		"Variable": "corporationToFactionID",
	})
	if err != nil {
		log.Fatal(err)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("format generated code: %v", err)
	}
	if *output == "" {
		if _, err := os.Stdout.Write(src); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := os.WriteFile(*output, src, 0o644); err != nil {
		log.Fatal(err)
	}
}
