
.PHONY: age-plugin-tkey
age-plugin-tkey: copy-deviceapp check-deviceapp-hash
	go build ./cmd/age-plugin-tkey

# TODO: probably something like this for release builds of tagged version:
#     go build -trimpath -buildvcs=false -ldflags="-X=main.version=$version"
# and CGO_ENABLED=0?

.PHONY: install
install:
	cp -af age-plugin-tkey /usr/local/bin/

.PHONY: copy-deviceapp
copy-deviceapp:
	cp -af ../tkey-device-x25519/x25519/app.bin internal/tkey/

.PHONY: check-deviceapp-hash
check-deviceapp-hash:
	@(cd internal/tkey; echo "file:$$(pwd)/app.bin hash:$$(sha512sum app.bin | cut -c1-16)… expected:$$(cut -c1-16 <app.bin.sha512)…"; sha512sum -cw app.bin.sha512)

.PHONY: clean
clean:
	rm -f age-plugin-tkey internal/tkey/app.bin

.PHONY: lint
lint:
	make -C gotools golangci-lint
	./gotools/golangci-lint run
