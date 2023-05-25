
.PHONY: build-in-container
build-in-container:
	podman image exists tkey-apps-builder || make build-image
	./build-in-container.sh

.PHONY: build-image
build-image:
	podman build -t localhost/tkey-apps-builder -f Containerfile
