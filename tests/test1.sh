#!/bin/bash
set -euo pipefail

mkdir -p out
rm -f out/test1-*

# default to NO touch required
notouch=--no-touch
if [[ "${1:-}" == "--touch" ]]; then
  notouch=
  shift
fi

export PATH=$(git rev-parse --show-toplevel):$PATH
export AGEDEBUG=plugin

age-plugin-tkey $notouch -g -o out/test1-id-tkey

r_tkey="$(sed -E -n "s/^# (recipient|public key): (age1.*)/\2/pi" out/test1-id-tkey)"

if [[ -z "$r_tkey" ]]; then
  printf "got no recipient\n"
  exit 1
fi

plaintext="too many secrets"
printf "%s" "$plaintext" | age -e -r "$r_tkey" -o out/test1-ciphertext

plaintextagain="$(age -d -i out/test1-id-tkey out/test1-ciphertext)"

if [[ "$plaintextagain" != "$plaintext" ]]; then
  printf "decrypt fail\n"
  exit 1
fi

exit 0
