package app

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEveNotificationTypes_ValidateCompletion(t *testing.T) {
	// validates that all types have been mapped to groups
	var allTypes []EveNotificationType
	for i := 1; i < int(numEveNotificationType); i++ {
		allTypes = append(allTypes, EveNotificationType(i))
	}
	mapped := slices.Collect(maps.Keys(notificationGroups))
	assert.ElementsMatch(t, allTypes, mapped)
}
