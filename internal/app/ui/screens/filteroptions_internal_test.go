package screens

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFixedFilterOptionsOrder pins the order of fixed choice lists that differs from alphabetical.
func TestFixedFilterOptionsOrder(t *testing.T) {
	cases := []struct {
		name string
		got  []string
		want []string
	}{
		{"asset search total", assetSearchTotalOptions(), []string{"Has total", "Has no total"}},
		{"augmentations implants", augmentationsImplantsOptions(), []string{"Has implants", "No implants"}},
		{"industry jobs activity", industryJobsActivityOptions(), []string{
			"Manufacturing",
			"Material efficiency research",
			"Time efficiency research",
			"Copying",
			"Invention",
			"Reactions",
		}},
		{"industry jobs installer", industryJobsInstallerOptions(), []string{"Installed by me", "Installed by corpmates"}},
		{"industry jobs owner", industryJobsOwnerOptions(), []string{"Owned by me", "Owned by corp"}},
		{"skill catalogue training", skillCatalogueTrainingOptions(), []string{
			"Trained",
			"Fully trained",
			"Queued",
			"Have prerequisites for",
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}
