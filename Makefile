
.PHONY: age-plugin-tkey
age-plugin-tkey: app
	go build ./cmd/age-plugin-tkey

.PHONY: install
install:
	cp -af age-plugin-tkey /usr/local/bin/

.PHONY: app
app:
	cp -af ../tkey-device-x25519/x25519/app.bin cmd/age-plugin-tkey/

.PHONY: clean
clean:
	rm -f age-plugin-tkey cmd/age-plugin-tkey/app.bin
