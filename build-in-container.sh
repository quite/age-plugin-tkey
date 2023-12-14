#!/bin/bash

builderimage=${1:-ghcr.io/quite/tkey-apps-builder:1}
printf "$0 using builderimage: %s\n" "$builderimage"

scriptf="$(mktemp)"

cat >"$scriptf" <<EOF
#!/bin/bash
set -eu

# remove what we'll produce (if successful)
rm -f /hostrepo/age-plugin-tkey
rm -f /hostrepo/internal/x25519.bin

mkdir /src
cp -af /hostrepo /src/age-plugin-tkey

cd /src/age-plugin-tkey
./build.sh

cp -afv age-plugin-tkey /hostrepo/age-plugin-tkey
cp -afv ./internal/tkey/x25519.bin /hostrepo/internal/tkey/x25519.bin
EOF

chmod +x "$scriptf"

cached=/tmp/age-plugin-tkey-build-cache-$UID
mkdir -p $cached/{go,.cache}

podman run --rm -i -t \
 --mount "type=bind,source=$(git rev-parse --show-toplevel),destination=/hostrepo" \
 --mount "type=bind,source=$scriptf,destination=/script.sh" \
 --mount "type=bind,source=$cached/go,destination=/root/go" \
 --mount "type=bind,source=$cached/.cache,destination=/root/.cache" \
 "$builderimage" \
 /script.sh

rm -f "$scriptf"
