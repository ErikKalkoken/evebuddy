// genschematicids generates a go source file containing a mapping for schematic IDs to icon IDs.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"maps"
	"os"
	"slices"
)

//go:embed target.go.template
var tmpl string

//go:embed iconids.json
var jsonData []byte

type row struct {
	SchematicID int
	IconID      int
}

var packageFlag = flag.String("p", "main", "package name")

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
	err = tmpl.Execute(os.Stdout, map[string]any{
		"Package":  *packageFlag,
		"Values":   values,
		"Variable": "schematicToIconIDs",
	})
	if err != nil {
		log.Fatal(err)
	}
}
