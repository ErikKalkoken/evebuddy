package character

import (
	"fmt"
	"strings"
)

type dotlanType = uint

const (
	dotlanAlliance dotlanType = iota
	dotlanCorporation
	dotlanSolarSystem
	dotlanRegion
)

func makeDotLanProfileURL(name string, typ dotlanType) string {
	const baseURL = "https://evemaps.dotlan.net"
	var path string
	m := map[dotlanType]string{
		dotlanAlliance:    "alliance",
		dotlanCorporation: "corp",
		dotlanSolarSystem: "system",
		dotlanRegion:      "region",
	}
	path, ok := m[typ]
	if !ok {
		return name
	}
	name2 := strings.ReplaceAll(name, " ", "_")
	return fmt.Sprintf("%s/%s/%s", baseURL, path, name2)
}
