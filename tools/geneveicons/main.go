// geneveicons generates a go source file containing a mapping for icon IDs to image files.
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
)

//go:embed target.go.template
var tmpl string

//go:embed data.json
var jsonData []byte

type row struct {
	ID  int
	Res string
}

var (
	packageFlag = flag.String("p", "main", "package name")
	output      = flag.String("out", "", "writes to given file when specified")
)

// Icon IDs to ignore, e.g. because we have no image file for it
var blacklistedIds = map[int]bool{21934: true}

func main() {
	flag.Parse()
	var data []map[string]any
	err := json.Unmarshal(jsonData, &data)
	if err != nil {
		log.Fatal(err)
	}
	var values []row
	for _, v := range data {
		id := int(v["id"].(float64))
		if blacklistedIds[id] {
			continue
		}
		s := v["file"].(string)
		s = strings.ReplaceAll(s, "_", "")
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, "png", "Png")
		r := row{ID: id, Res: "resource" + s}
		values = append(values, r)
	}
	tmpl, err := template.New("").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}
	var w io.Writer
	if *output == "" {
		w = os.Stdout
	} else {
		f, err := os.Create(*output)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		w = f
	}
	err = tmpl.Execute(w, map[string]any{
		"Package":  *packageFlag,
		"Values":   values,
		"Variable": "id2fileMap",
	})
	if err != nil {
		log.Fatal(err)
	}
}
