# spellchecker: disable
generate: bundle queries mapping

bundle:
	fyne bundle --package icons --prefix "" resources/app > internal/app/icons/resource.go
	fyne bundle --package widget --prefix icon resources/widget > internal/widget/resource.go
	fyne bundle --package eveimageservice resources/eveimageservice > internal/eveimageservice/resource.go
	fyne bundle --package eveicon resources/eveicon > internal/eveicon/resource.go

mapping:
	go run ./tools/geneveicons/ -p eveicon > internal/eveicon/mapping.go
	gofmt -s -w internal/eveicon/mapping.go
	go run ./tools/genschematicids/ -p eveicon > internal/eveicon/schematic.go
	gofmt -s -w internal/eveicon/schematic.go
	go run ./tools/genratelimit/ -p xesi > internal/xesi/ratelimit_gen.go

queries:
	sqlc generate

build-appimage:
	tools/build_appimage.sh

release:
	fyne package --os linux --release --tags migrated_fynedo

appimage:
	release build-appimage

loc:
	gocloc ./internal --by-file --include-lang=Go --not-match="\.sql\.go" --not-match-d="eveicon" --not-match="_test\.go"

deploy-android: check-device build-android install-android

build-android:
	fyne package -os android --release --tags migrated_fynedo

install-android:
	adb install -r -d EVE_Buddy.apk

# check-device aborts when no Android device is connected
check-device:
	@if ! adb devices | grep -q device$$; then\
		echo "device not found";\
		exit 1;\
	fi
	@echo device found


interfaces:
	ifacemaker -f internal/eveimageservice/eveimageservice.go -i EveImageService -p app -s EveImageService > internal/app/eveimageservice.go

test_races:
	GORACE="log_path=.temp/datarace.txt halt_on_error=1" go run -race --tags migrated_fynedo .

build:
	fyne build --tags migrated_fynedo
