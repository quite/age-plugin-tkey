#!/bin/bash

builderimage=${1:-tkey-apps-builder}
printf "$0 using builderimage: %s\n" "$builderimage"

cd "${0%/*}" || exit 1
cd ..

scriptf="$(mktemp)"

cat >"$scriptf" <<EOF
#!/bin/bash

clang --version

mkdir /src
# Remove what we'll produce (if successful)
rm -f /age-plugin-tkey/age-plugin-tkey
rm -f /age-plugin-tkey/internal/x25519.bin

cp -af /age-plugin-tkey /src/

git -C /src clone --depth=1 --branch=v0.0.2 https://github.com/tillitis/tkey-libs
make -C /src/tkey-libs -j

# TODO change once tagged
git -C /src clone --depth=1 --branch=main https://github.com/quite/tkey-device-x25519
make -C /src/tkey-device-x25519 x25519/app.bin

make -C /src/age-plugin-tkey clean
make -C /src/age-plugin-tkey -j age-plugin-tkey

cp -af /src/age-plugin-tkey/age-plugin-tkey /age-plugin-tkey/age-plugin-tkey
cp -af /src/tkey-device-x25519/x25519/app.bin /age-plugin-tkey/internal/tkey/x25519.bin
EOF

chmod +x "$scriptf"

cached=/tmp/age-plugin-tkey-build-cache-$UID
mkdir -p $cached/{go,.cache}

podman run --rm -i -t \
 --mount "type=bind,source=$(pwd),destination=/age-plugin-tkey" \
 --mount "type=bind,source=$scriptf,destination=/inside.sh" \
 --mount "type=bind,source=$cached/go,destination=/root/go" \
 --mount "type=bind,source=$cached/.cache,destination=/root/.cache" \
 $builderimage \
 /inside.sh

rm -f "$scriptf"

ls -l "$(realpath ./age-plugin-tkey)" "$(realpath ./internal/tkey/x25519.bin)"
