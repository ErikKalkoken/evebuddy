help:
	@echo "Make file for EVE Buddy"

sqlgen:
	sqlc generate

images:
	fyne bundle --package ui resources/images > internal/ui/resource.go

icons:
	fyne bundle --package icons resources/icons > internal/eveonline/icons/resource.go
	python3 tools/icons_map/generate.py > internal/eveonline/icons/mapping.go
