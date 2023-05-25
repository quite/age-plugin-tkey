#!/bin/bash

cd "${0%/*}" || exit 1

scriptf="$(mktemp)"

cat >"$scriptf" <<EOF
#!/bin/bash
mkdir -p /src
cd /src
cp -af /age-plugin-tkey .
git clone --depth=1 --branch=v0.0.1 https://github.com/tillitis/tkey-libs
git clone --depth=1 https://github.com/quite/tkey-device-x25519
make -C /src/tkey-libs -j
make -C /src/tkey-device-x25519 -j
make -C /src/age-plugin-tkey clean
make -C /src/age-plugin-tkey -j age-plugin-tkey
cp -af /src/age-plugin-tkey/age-plugin-tkey /age-plugin-tkey/
EOF

chmod +x "$scriptf"

cached=/tmp/age-plugin-tkey-build-cache
mkdir -p $cached/{go,dotcache}

podman run --rm -it \
       --mount type=bind,source=$PWD/..,destination=/age-plugin-tkey \
       --mount type=bind,source=$scriptf,destination=/inside.sh \
       --mount type=bind,source=$cached/go,destination=/root/go \
       --mount type=bind,source=$cached/dotcache,destination=/root/.cache \
       age-plugin-tkey-builder \
       /inside.sh

rm -f "$scriptf"
