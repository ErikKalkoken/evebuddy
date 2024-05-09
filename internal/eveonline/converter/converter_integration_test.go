package converter_test

import (
	"fmt"
	"testing"

	"github.com/ErikKalkoken/evebuddy/internal/eveonline/converter"
	"github.com/stretchr/testify/assert"
)

func TestXMLtoMarkdown(t *testing.T) {
	var testCases = []struct {
		in   string
		want string
	}{
		{
			"simple",
			"simple",
		},
		{
			"s<loc>imp</loc>le",
			"simple",
		},
		{
			`<a href="showinfo:1376//93330670">Erik</a>`,
			"[Erik](https://zkillboard.com/character/93330670/)",
		},
		{
			`<a href="showinfo:2//98267621">Congregation</a>`,
			"[Congregation](https://zkillboard.com/corporation/98267621/)",
		},
		{
			`<a href="showinfo:16159//99005502">No Handlebars.</a>`,
			"[No Handlebars.](https://zkillboard.com/alliance/99005502/)",
		},
		{
			`<a href="showinfo:5//30004984">Abune</a>`,
			"[Abune](https://zkillboard.com/system/30004984/)",
		},
		{
			`<a href="showinfo:52678//60003760">jita</a>`,
			"jita",
		},
		{
			`<a href="showinfo:35834//1022167642188">Amamake - 3 Time Nearly AT Winners</a>`,
			"Amamake - 3 Time Nearly AT Winners",
		},
		{
			`<a href="killReport:84900666:9e6fe9e5392ff0cfc6ab956677dbe1deb69c4b04">Kill: Yuna Kobayashi (Badger)</a>`,
			"Kill: Yuna Kobayashi (Badger)",
		},
		{
			`<a href="fitting:78366:2048;1:13001;1:2410;6:35660;1:31724;1:3568;1:31796;2:4405;3:31932;2:4349;2:21640;2:2185;4:21638;2:2175;2:2629;1500:32006;19:27435;1000:27441;1000:27447;1000:27453;1000:24511;1500::">roamy</a>`,
			"roamy",
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("in: %s out: %s", tc.in, tc.want), func(t *testing.T) {
			got := converter.XMLtoMarkdown(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}

}
