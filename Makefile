help:
	@echo "Make file for EVE Buddy"

static:
	fyne bundle --package ui images > internal/ui/resource.go