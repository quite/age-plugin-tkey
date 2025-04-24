# v0.0.6

- Solve issue with non-X25519 stanzas https://github.com/quite/age-plugin-tkey/issues/5
- Bump go dependency filippo.io/age

tkey-device-x25519: v0.0.2 -- identity not changed

# v0.0.5

- Bump go dependencies only (in particular filippo.io/age v1.2.0)

tkey-device-x25519: v0.0.2 -- identity not changed

# v0.0.4

- Bump go deps

tkey-device-x25519: v0.0.2 -- identity not changed

# v0.0.3

- Add -y flag to convert TKey identities to recipients.

# v0.0.2

- Add TKey serial number (UDI) as a comment in the output. Only
  possible when we load the app ourselves.
- Make a small optimization that avoids unwrapping the file key a 2nd
  time if it was already unwrapped once.

tkey-device-x25519: v0.0.1 -- identity not changed

# v0.0.1

The first tagged release. It uses tkey-device-x25519 built from tag
v0.0.1.
