// genplanetschematics generates a go source file containing the inputs and outputs
// of planetary industry schematics.
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

//go:embed planetSchematics.yaml
var yamlData []byte

type schematic struct {
	CycleTime int64                   `yaml:"cycleTime"`
	Types     map[int64]schematicType `yaml:"types"`
}

type schematicType struct {
	IsInput  bool  `yaml:"isInput"`
	Quantity int64 `yaml:"quantity"`
}

type input struct {
	TypeID   int64
	Quantity int64
}

type row struct {
	ID             int64
	CycleTime      int64
	Inputs         []input
	OutputTypeID   int64
	OutputQuantity int64
}

var (
	packageFlag = flag.String("p", "main", "package name")
	output      = flag.String("out", "", "writes to given file when specified")
)

func main() {
	flag.Parse()
	var data map[int64]schematic
	err := yaml.Unmarshal(yamlData, &data)
	if err != nil {
		log.Fatal(err)
	}

	var values []row
	for id, s := range data {
		r := row{ID: id, CycleTime: s.CycleTime}
		var outputs int
		for typeID, t := range s.Types {
			if t.IsInput {
				r.Inputs = append(r.Inputs, input{TypeID: typeID, Quantity: t.Quantity})
				continue
			}
			r.OutputTypeID = typeID
			r.OutputQuantity = t.Quantity
			outputs++
		}
		if outputs != 1 {
			log.Fatalf("schematic %d: expected 1 output, got %d", id, outputs)
		}
		slices.SortFunc(r.Inputs, func(a, b input) int {
			return cmp.Compare(a.TypeID, b.TypeID)
		})
		values = append(values, r)
	}
	slices.SortFunc(values, func(a, b row) int {
		return cmp.Compare(a.ID, b.ID)
	})
	tmpl, err := template.New("").Parse(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		"Package":  *packageFlag,
		"Values":   values,
		"Variable": "planetSchematics",
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
