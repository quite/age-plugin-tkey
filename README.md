
Work In Progress

Plugin for [age](https://github.com/FiloSottile/age) to use a Tillitis
TKey USB security key.

It embeds and uses the TKey device app
[tkey-device-x25519](https://github.com/quite/tkey-device-x25519) for
doing X25519 ECDH. For communicating with the device app running on
the TKey it uses [tkeyx25519](https://github.com/quite/tkeyx25519).

Note that this is work in progress, in particular the
tkey-device-x25519 is not yet considered stable. The implementation
may change, and this will cause a change of identity of a TKey running
it. This would mean that the public/private key no longer is the same,
and decryption of data encrypted for the previous key pair will not be
possible.

## Usage

Here we create a new keypair/identity and learn about the public
key/recipient that is us. Then we encrypt a note to self, and proceed
to decrypt it.

```
$ age-plugin-tkey --generate >my-keys
# recipient: age1xuqv8tq5ttkgwe3quys0dfwxv6zzqpemvckjeutudtjjhfac2f9q6lc377
# touchRequired: false
$ echo "remember to fix all bugs!" | age --encrypt -a -r age1xuqv8tq5ttkgwe3quys0dfwxv6zzqpemvckjeutudtjjhfac2f9q6lc377 >note-to-self
$ age -i my-keys --decrypt ./note-to-self
remember to fix all bugs!
```

To create an identity which requires a physical touch of TKey upon
ECDH key exchange, add the flag `--touch` when generating.

The file `my-keys` ends up containing a line beginning with
`AGE-PLUGIN-TKEY-` which holds the parameters used for deriving the
secret key on the TKey. The secret key itself never leaves the TKey
hardware.

To run towards an emulated TKey in QEMU instead of real hardware, you
can set the environment variable `TKEY_PORT` to QEMU's char-device
before running age, like `TKEY_PORT=/dev/pts/22`.

## Building

You first need to build the device app in a sibling directory. Do `git
-C .. clone https://github.com/quite/tkey-device-x25519` and follow
the instructions in the README.md there. Then just `make` here.

For reproducability and maintaining a stable device app hash and thus
identity we typically build in a container image. There is some stuff
in [contrib/](contrib/) for doing that using podman, try `make`.
