
.PHONY: age-plugin-tkey
age-plugin-tkey: check-deviceapp-hashes
	go build ./cmd/age-plugin-tkey

# TODO: probably something like this for release builds of tagged version:
#     go build -trimpath -buildvcs=false -ldflags="-X=main.version=$version"
# and CGO_ENABLED=0?

.PHONY: install
install:
	cp -af age-plugin-tkey /usr/local/bin/

.PHONY: check-deviceapp-hashes
check-deviceapp-hashes:
	(cd internal/tkey && sha512sum -c -w --ignore-missing x25519-hashes.sha512)

.PHONY: clean
clean:
	rm -f age-plugin-tkey


.PHONY: build-in-container
build-in-container:
	podman image exists tkey-apps-builder || make build-image
	./build-in-container.sh tkey-apps-builder
.PHONY: build-image
build-image:
	#--pull=always --no-cache
	podman build -t localhost/tkey-apps-builder -f Containerfile


golangci_version=$(shell grep -A2 "uses: golangci" .github/workflows/ci.yaml | grep -o -m1 "v[0-9]\+\.[.0-9]\+")
golangci_cachedir=$(HOME)/.cache/golangci-lint/$(golangci_version)
.PHONY: lint
lint:
	mkdir -p $(golangci_cachedir)
	podman run --rm -it \
		-v $$(pwd):/src -w /src \
		-v $(golangci_cachedir):/root/.cache \
		docker.io/golangci/golangci-lint:$(golangci_version)-alpine \
		golangci-lint run
