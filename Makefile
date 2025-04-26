generate: bundle queries mapping

bundle:
	fyne bundle --package icons --prefix "" resources/app > internal/app/icons/resource.go
	fyne bundle --package widget --prefix icon resources/widget > internal/widget/resource.go
	fyne bundle --package eveimageservice resources/eveimageservice > internal/eveimageservice/resource.go
	fyne bundle --package eveicon resources/eveicon > internal/eveicon/resource.go

mapping:
	python3 tools/icons_map/generate.py > internal/eveicon/mapping.go
	go run ./tools/genschematicids/ -p eveicon > internal/eveicon/schematic.go

queries:
	sqlc generate

appimage:
	tools/build_appimage.sh

release:
	fyne package --os linux --release

loc:
	gocloc ./internal --by-file --include-lang=Go --not-match="\.sql\.go" --not-match-d="eveicon" --not-match="_test\.go"

deploy-android:
	fyne package -os android
	adb install -r -d EVE_Buddy.apk

install-android:
	adb install -r -d EVE_Buddy.apk

interface_settings:
	ifacemaker -s Settings -i Settings -p app -f internal/app/settings/settings.go -o internal/app/settings.go

interfaces:
	interface_settings
	ifacemaker -s CharacterService -i CharacterService -p app -o internal/app/characterservice.go -f internal/app/characterservice/characterservice.go
