#!/bin/bash

mkdir -p out
rm -f out/test2-*

# default to NO touch required
notouch=--no-touch
if [[ "${1:-}" == "--touch" ]]; then
  notouch=
  shift
fi

export PATH=$(git rev-parse --show-toplevel):$PATH
export AGEDEBUG=plugin

age-plugin-tkey $notouch -g -o out/test2-id-tkey-only
age-keygen -o out/test2-id-age-only

r_tkey="$(sed -E -n "s/^# (recipient|public key): (age1.*)/\2/pi" out/test2-id-tkey-only)"
r_age="$(sed -E -n "s/^# (recipient|public key): (age1.*)/\2/pi" out/test2-id-age-only)"

cat >out/test2-id-tkey-and-age out/test2-id-tkey-only out/test2-id-age-only

if [[ -z "$r_tkey" ]] || [[ -z "$r_age" ]]; then
  printf "did not get both recipients\n"
  exit 1
fi

plaintext="too many secrets"
printf "%s" "$plaintext" | age -e -r "$r_age" -r "$r_tkey" -o out/test2-ciphertext-both

cat <<EOF

# Now we have a ciphertext encrypted to BOTH an age identity on disk,
# and to identity on TKey. Try these:

# Should decrypt, both with and without TKey plugged in.
age -d -i out/test2-id-tkey-and-age <out/test2-ciphertext-both

# Should decrypt only with TKey plugged in
age -d -i out/test2-id-tkey-only <out/test2-ciphertext-both

# Should decrypt, no matter any TKey
age -d -i out/test2-id-age-only <out/test2-ciphertext-both

# If you ran test2.sh --touch and try to decrypt with the TKey
# identity, then you have to touch the TKey 2 times. Could this be
# because recipients for age-plugin-tkey identities look just like the
# age native recipients? Because it has an ed25519 pubkey. So both
# have to be tried by the plugin? Hm, but adding a 3rd (age native)
# identity to encrypt for does not result in the need for touch 3
# times... (TODO).

EOF

exit 0
