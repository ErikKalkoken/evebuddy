help:
	@echo "Make file for EVE Buddy"

build_icons:
	fyne bundle --package ui icons > internal/ui/icons.go