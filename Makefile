
.PHONY: age-plugin-tkey
age-plugin-tkey: copy-deviceapp check-deviceapp-hash
	go build ./cmd/age-plugin-tkey

# TODO: probably something like this for release builds of tagged version:
# go build -trimpath -buildvcs=false -ldflags="-X=main.version=$version"

.PHONY: install
install:
	cp -af age-plugin-tkey /usr/local/bin/

.PHONY: copy-deviceapp
copy-deviceapp:
	cp -af ../tkey-device-x25519/x25519/app.bin internal/tkey/

.PHONY: check-deviceapp-hash
check-deviceapp-hash:
	cd internal/tkey && sha512sum -c app.bin.sha512

.PHONY: clean
clean:
	rm -f age-plugin-tkey internal/tkey/app.bin

.PHONY: lint
lint:
	make -C gotools golangci-lint
	./gotools/golangci-lint run
