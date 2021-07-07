
.PHONY: build
build: clean
	GOOS=darwin GOARCH=amd64 go build -o OncallStatus.app/Contents/OncallStatus .

.PHONY: sign
sign:
	@test -n "$(AC_USERNAME)" || (echo "[ERROR] AC_USERNAME is not set" && exit 1)
	@test -n "$(AC_PASSWORD)" || (echo "[ERROR] AC_PASSWORD is not set" && exit 1)
	@test -f OncallStatus.app/Contents/OncallStatus || (echo "[ERROR] Application binary does not exist" && exit 1)
	gon -log-level=info ./gon.hcl

.PHONY: clean
clean:
	rm -rf OncallStatus.app/Contents/OncallStatus \
	  OncallStatus.app/Contents/_CodeSignature/ \
	  oncall-status.dmg
