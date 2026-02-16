// gennpccorps generates a go source file containing additional information
// about NPC corporations.
package main

import (
	_ "embed"
	"flag"
	"html/template"
	"io"
	"log"
	"os"

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
	tmpl, err := template.New("").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
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
	err = tmpl.Execute(out, map[string]any{
		"Package":  *packageFlag,
		"Values":   values,
		"Variable": "corporationToFactionID",
	})
	if err != nil {
		log.Fatal(err)
	}
}
