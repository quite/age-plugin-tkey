#!/bin/bash

cd "${0%/*}" || exit 1
cd ..

scriptf="$(mktemp)"

cat >"$scriptf" <<EOF
#!/bin/bash
mkdir /src
cp -af /age-plugin-tkey /src/
# TODO change to a tag > v0.0.1 when ready
git -C /src clone --depth=1 --branch=main https://github.com/tillitis/tkey-libs
# TODO change to main once merged
git -C /src clone --depth=1 --branch=rework https://github.com/quite/tkey-device-x25519
make -C /src/tkey-libs -j
make -C /src/tkey-device-x25519 -j
make -C /src/age-plugin-tkey clean
make -C /src/age-plugin-tkey -j age-plugin-tkey
cp -af /src/age-plugin-tkey/age-plugin-tkey /age-plugin-tkey/
cp -af /src/tkey-device-x25519/x25519/app.bin /age-plugin-tkey/internal/tkey/
EOF

chmod +x "$scriptf"

cached=/tmp/age-plugin-tkey-build-cache-$UID
mkdir -p $cached/{go,.cache}

podman run --rm -i -t \
 --mount "type=bind,source=$PWD,destination=/age-plugin-tkey" \
 --mount "type=bind,source=$scriptf,destination=/inside.sh" \
 --mount "type=bind,source=$cached/go,destination=/root/go" \
 --mount "type=bind,source=$cached/.cache,destination=/root/.cache" \
 tkey-apps-builder \
 /inside.sh

rm -f "$scriptf"

ls -l "$(realpath ./age-plugin-tkey)" "$(realpath ./internal/tkey/app.bin)"
