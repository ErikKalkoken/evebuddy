# spellchecker: disable
generate: bundle queries mapping

bundle:
	go generate ./internal/eveicon ;
	go generate ./internal/eveimageservice ;
	go generate ./internal/app/icons ;

mapping:
	go generate ./internal/eveicon ;
	go generate ./internal/xgoesi ;

queries:
	sqlc generate

build-appimage:
	tools/build_appimage.sh

release:
	fyne package --os linux --release --tags migrated_fynedo

appimage: release build-appimage

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
	GORACE="halt_on_error=1" go run -race --tags migrated_fynedo .
# 	GORACE="log_path=.temp/datarace.txt halt_on_error=1" go run -race --tags migrated_fynedo . -log-level DEBUG

build:
	fyne build --tags migrated_fynedo

ratelimitdoc:
	go run ./tools/genratelimit/ -f md  > ratelimits.md