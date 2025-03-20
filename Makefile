generate: bundle queries

bundle:
	fyne bundle --package icons --prefix "" resources/app > internal/app/icons/resource.go
	fyne bundle --package widget --prefix icon resources/widget > internal/widget/resource.go
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

deploy-android:
	fyne package -os android
	adb install -r -d EVE_Buddy.apk

install-android:
	adb install -r -d EVE_Buddy.apk

interface:
	ifacemaker -s BaseUI -i UI -p app -f internal/app/ui/ui.go

settings:
	ifacemaker -s AppSettings -i Settings -p app -f internal/app/ui/appsettings.go -o internal/app/settings.go
