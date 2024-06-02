help:
	@echo "Make file for EVE Buddy"

sqlgen:
	sqlc generate

static:
	fyne bundle --package ui images > internal/ui/resource.go