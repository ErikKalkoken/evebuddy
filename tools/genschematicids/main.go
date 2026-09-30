// genschematicids generates a go source file containing a mapping for schematic IDs to icon IDs.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"go/format"
	"log"
	"maps"
	"os"
	"slices"
	"text/template"
)

//go:embed target.go.template
var tmpl string

//go:embed iconids.json
var jsonData []byte

type row struct {
	SchematicID int
	IconID      int
}

var (
	packageFlag = flag.String("p", "main", "package name")
	output      = flag.String("out", "", "writes to given file when specified")
)

func main() {
	flag.Parse()
	var v map[string][]map[string]int
	err := json.Unmarshal(jsonData, &v)
	if err != nil {
		log.Fatal(err)
	}
	k := slices.Collect(maps.Keys(v))[0]
	v2 := v[k]
	var values []row
	for _, v := range v2 {
		r := row{SchematicID: v["schematicID"], IconID: v["iconID"]}
		values = append(values, r)
	}
	tmpl, err := template.New("").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		"Package":  *packageFlag,
		"Values":   values,
		"Variable": "schematicToIconIDs",
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
