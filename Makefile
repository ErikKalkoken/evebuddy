generate: bundle queries

bundle:
	fyne bundle --package ui --prefix Icon resources/ui > internal/app/ui/icon.go
	fyne bundle --package widgets --prefix icon resources/widgets > internal/app/widgets/resource.go
	fyne bundle --package eveimage resources/eveimage > internal/eveimage/resource.go
	fyne bundle --package eveicon resources/eveicon > internal/eveicon/resource.go
	python3 tools/icons_map/generate.py > internal/eveicon/mapping.go

queries:
	sqlc generate

appimage:
	tools/build_appimage.sh

release:
	fyne package --os linux --release

loc:
	gocloc ./internal --by-file --include-lang=Go --not-match="\.sql\.go" --not-match-d="eveicon" --not-match="_test\.go"
