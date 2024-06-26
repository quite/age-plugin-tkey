#!/bin/bash

# investigate https://github.com/str4d/rage/issues/414

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
# encrypting with the tkey-recipient first, avoiding having to touch 2 times
printf "%s" "$plaintext" | age -e -r "$r_tkey" -r "$r_age" -o out/test2-ciphertext-both

cat <<EOF

# Now we have a ciphertext encrypted to BOTH an age identity on disk,
# and to identity on TKey. Here follows some tests you can try.

# First you may set PATH to newly built age-plugin-tkey
export PATH=$(git rev-parse --show-toplevel):\$PATH
# Maybe enable debug
export AGEDEBUG=plugin

# The following should decrypt, both with and without TKey plugged in.
age -d -i out/test2-id-tkey-and-age <out/test2-ciphertext-both

# The following should only decrypt with TKey plugged in
age -d -i out/test2-id-tkey-only <out/test2-ciphertext-both

# The following should decrypt, no matter any TKey
age -d -i out/test2-id-age-only <out/test2-ciphertext-both
EOF

exit 0
