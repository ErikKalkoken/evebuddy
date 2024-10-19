bundle:
	fyne bundle --package ui resources/ui > internal/app/ui/resource.go
	fyne bundle --package eveimage resources/eveimage > internal/eveimage/resource.go
	fyne bundle --package eveicon resources/eveicon > internal/eveicon/resource.go
	python3 tools/icons_map/generate.py > internal/eveicon/mapping.go

queries:
	sqlc generate

appimage:
	tools/build_appimage.sh

release:
	fyne package --os linux --release