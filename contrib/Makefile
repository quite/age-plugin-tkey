builderimage=tkey-apps-builder

.PHONY: build-in-container
build-in-container:
	podman image exists $(builderimage) || make build-image
	./build-in-container.sh $(builderimage)

.PHONY: build-image
build-image:
	# --pull=always --no-cache
	podman build -t localhost/$(builderimage) -f Containerfile
