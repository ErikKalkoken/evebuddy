help:
	@echo "Make file for EVE Buddy"

sqlgen:
	sqlc generate

images:
	fyne bundle --package ui resources/images > internal/ui/resource.go