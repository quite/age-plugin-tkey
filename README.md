
Work In Progress

Plugin for [age](https://github.com/FiloSottile/age) to use a Tillitis
TKey USB security key.

It embeds and uses the TKey device app
[tkey-device-x25519](https://github.com/quite/tkey-device-x25519) for
doing X25519 ECDH. For communicating with the device app running on
the TKey it uses [tkeyx25519](https://github.com/quite/tkeyx25519).

# Building

You first need to build the device app in a sibling directory. Do `git
-C .. clone https://github.com/quite/tkey-device-x25519` and follow
the instructions in the README.md there. Then just `make` here.

For reproducability and maintaining a stable device app hash and thus
identity, there is some stuff in [contrib/](contrib/) for building
using a container image.
