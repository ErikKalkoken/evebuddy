help:
	@echo "Make file for Eve Buddy"

build_icons:
	fyne bundle --package ui icons > internal/ui/icons.go