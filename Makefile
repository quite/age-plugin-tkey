
.PHONY: age-plugin-tkey
age-plugin-tkey: app
	go build ./cmd/age-plugin-tkey

.PHONY: app
app:
	# TODO should it really make it itself?
	make -C ../tkey-device-x25519 x25519/app.bin
	cp -af ../tkey-device-x25519/x25519/app.bin cmd/age-plugin-tkey/

.PHONY: clean
clean:
	rm -f age-plugin-tkey cmd/age-plugin-tkey/app.bin
