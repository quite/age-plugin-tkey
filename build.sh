#!/bin/bash
set -eu

clang --version

LIBS="v0.0.2"
# TODO change once tagged
X25519="main"

LIBSD=../tkey-libs
X25519D=../tkey-device-x25519

if [[ ! -e $LIBSD ]]; then
  git clone --branch=$LIBS https://github.com/tillitis/tkey-libs $LIBSD
else
  printf "NOTE: building with existing %s, possibly not a clean clone!\n" $LIBSD
fi
(cd $LIBSD; pwd; git describe --dirty --long --always)
make -C $LIBSD -j

if [[ ! -e $X25519D ]]; then
  git clone --branch=$X25519 --depth=1 https://github.com/quite/tkey-device-x25519 $X25519D
else
  printf "NOTE: building with existing %s, possibly not a clean clone!\n" $X25519D
fi
(cd $X25519D; pwd; git describe --dirty --long --always)
# skipping the hash check
make -C $X25519D -j x25519/app.bin

make clean

cp -afv $X25519D/x25519/app.bin ./internal/tkey/x25519.bin
make -j
