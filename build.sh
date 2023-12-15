#!/bin/bash
set -eu

clang --version

TAGLIBS="v0.0.2"
TAGX25519="v0.0.1"

DIRLIBS=../tkey-libs
DIRX25519=../tkey-device-x25519

if [[ ! -e $DIRLIBS ]]; then
  git clone --branch=$TAGLIBS https://github.com/tillitis/tkey-libs $DIRLIBS
else
  printf "NOTE: building with existing %s, possibly not a clean clone!\n" $DIRLIBS
fi
make -C $DIRLIBS -j

if [[ ! -e $DIRX25519 ]]; then
  git clone --branch=$TAGX25519 --depth=1 https://github.com/quite/tkey-device-x25519 $DIRX25519
else
  printf "NOTE: building with existing %s, possibly not a clean clone!\n" $DIRX25519
fi
# skipping the hash check
make -C $DIRX25519 -j x25519/app.bin

make clean

printf "DEPENDENCY: %-22s %-7s git-describe: %s\n" $DIRLIBS $TAGLIBS "$(git -C $DIRLIBS describe --dirty --long --always)"
printf "DEPENDENCY: %-22s %-7s git-describe: %s\n" $DIRX25519 $TAGX25519 "$(git -C $DIRX25519 describe --dirty --long --always)"

cp -afv $DIRX25519/x25519/app.bin ./internal/tkey/x25519-$TAGX25519.bin
make -j
