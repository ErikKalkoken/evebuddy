generate: bundle queries mapping

bundle:
	fyne bundle --package icons --prefix "" resources/app > internal/app/icons/resource.go
	fyne bundle --package widget --prefix icon resources/widget > internal/widget/resource.go
	fyne bundle --package eveimageservice resources/eveimageservice > internal/eveimageservice/resource.go
	fyne bundle --package eveicon resources/eveicon > internal/eveicon/resource.go

mapping:
	go run ./tools/geneveicons/ -p eveicon > internal/eveicon/mapping.go
	go run ./tools/genschematicids/ -p eveicon > internal/eveicon/schematic.go

queries:
	sqlc generate

build-appimage:
	tools/build_appimage.sh

release:
	fyne package --os linux --release --tags migrated_fynedo

appimage: release build-appimage

loc:
	gocloc ./internal --by-file --include-lang=Go --not-match="\.sql\.go" --not-match-d="eveicon" --not-match="_test\.go"

deploy-android: check-device make-android install-android

make-android:
	fyne package -os android

install-android:
	adb install -r -d EVE_Buddy.apk

# check-device aborts when no Android device is connected
check-device:
	@if ! adb devices | grep -q device$$; then\
		echo "device not found";\
		exit 1;\
	fi
	@echo device found