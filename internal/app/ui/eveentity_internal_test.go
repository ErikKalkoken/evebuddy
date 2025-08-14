package ui

// func TestEntityIcon(t *testing.T) {
// 	cases := []struct {
// 		id        int32
// 		category  string
// 		want      []byte
// 		wantError bool
// 	}{
// 		{42, "alliance", alliance, false},
// 		{42, "character", character, false},
// 		{42, "corporation", corporation, false},
// 		{888, "faction", faction, false},
// 		{42, "inventory_type", typ, false},
// 		{1, "invalid", nil, true},
// 	}
// 	for _, tc := range cases {
// 		t.Run(tc.category, func(t *testing.T) {
// 			c.Clear()
// 			r, err := m.EntityIcon(tc.id, tc.category, 64)
// 			if !tc.wantError {
// 				if assert.NoError(t, err) {
// 					got := r.Content()
// 					assert.Equal(t, tc.want, got)
// 				}
// 			} else {
// 				assert.Error(t, err)
// 			}
// 		})
// 	}
// }
