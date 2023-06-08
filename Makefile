
.PHONY: age-plugin-tkey
age-plugin-tkey: copy-deviceapp check-deviceapp-hash
	go build ./cmd/age-plugin-tkey

.PHONY: install
install:
	cp -af age-plugin-tkey /usr/local/bin/

.PHONY: copy-deviceapp
copy-deviceapp:
	cp -af ../tkey-device-x25519/x25519/app.bin cmd/age-plugin-tkey/

.PHONY: check-deviceapp-hash
check-deviceapp-hash:
	cd cmd/age-plugin-tkey && sha512sum -c app.bin.sha512

.PHONY: clean
clean:
	rm -f age-plugin-tkey cmd/age-plugin-tkey/app.bin
