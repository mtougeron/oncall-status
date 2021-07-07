
.PHONY: build
build: clean
	GOOS=darwin GOARCH=amd64 go build -o OncallStatus.app/Contents/OncallStatus .

.PHONY: dmg
dmg:
	create-dmg \
	  --volname "oncall-status" \
	  --window-pos 200 120 \
	  --window-size 800 400 \
	  --icon-size 100 \
	  --icon "OncallStatus.app" 200 190 \
	  --hide-extension "OncallStatus.app" \
	  --app-drop-link 600 185 \
	  "oncall-status.dmg" \
	  "OncallStatus.app"

.PHONY: sign
sign: signapp signdmg

.PHONY: sign-app
sign-app:
	@test -n "$(AC_USERNAME)" || (echo "[ERROR] AC_USERNAME is not set" && exit 1)
	@test -n "$(AC_PASSWORD)" || (echo "[ERROR] AC_PASSWORD is not set" && exit 1)
	@test -f OncallStatus.app/Contents/OncallStatus || (echo "[ERROR] Application binary does not exist" && exit 1)
	gon -log-level=info ./gon-app.hcl

.PHONY: notarize-dmg
notarize-dmg:
	@test -n "$(AC_USERNAME)" || (echo "[ERROR] AC_USERNAME is not set" && exit 1)
	@test -n "$(AC_PASSWORD)" || (echo "[ERROR] AC_PASSWORD is not set" && exit 1)
	@test -f oncall-status.dmg || (echo "[ERROR] oncall-status.dmg does not exist" && exit 1)
	gon -log-level=info ./gon-dmg.hcl

.PHONY: clean
clean:
	rm -rf OncallStatus.app/Contents/OncallStatus \
	  OncallStatus.app/Contents/_CodeSignature/ \
	  oncall-status.dmg
