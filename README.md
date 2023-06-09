
Work In Progress

Plugin for [age](https://github.com/FiloSottile/age) to use a Tillitis
[TKey](https://github.com/tillitis/tillitis-key1) USB security key.

For doing X25519 ECDH it embeds and runs the device app
[tkey-device-x25519](https://github.com/quite/tkey-device-x25519) on
the TKey. The Go package
[tkeyx25519](https://github.com/quite/tkeyx25519) is used for
communicating with this device app.

Note that this is work in progress. In particular, tkey-device-x25519
is not yet considered stable. The implementation may change, which
would change the identity of a TKey running it. This would mean that
the public/private key no longer is the same and that decryption of
data encrypted for the previous key pair will be impossible.

## Usage

In the following we create a new keypair/identity and learn about the
public key/recipient that is us. Then we encrypt a note to ourselves,
and proceed to decrypt it.

```
$ age-plugin-tkey --generate >my-keys
# recipient: age1xuqv8tq5ttkgwe3quys0dfwxv6zzqpemvckjeutudtjjhfac2f9q6lc377
$ echo "remember to fix all bugs!" | age --encrypt -a -r age1xuqv8tq5ttkgwe3quys0dfwxv6zzqpemvckjeutudtjjhfac2f9q6lc377 >note-to-self
$ age -i my-keys --decrypt ./note-to-self
remember to fix all bugs!
```

To create an identity which requires a physical touch of TKey upon
ECDH key exchange, add the flag `--touch` when generating.

After running the above, te file `my-keys` ends up containing a line
beginning with `AGE-PLUGIN-TKEY-`. This holds the parameters used for
deriving the secret key on the TKey (which must be the exact same
physical device as used for generation). The secret key itself never
leaves the TKey hardware, which actually has no storage. You can learn
more about this here:
https://dev.tillitis.se/intro/#measured-boot--secrets

To run the plugin towards an emulated TKey in QEMU, instead of real
hardware, you can set the environment variable `TKEY_PORT` to QEMU's
char-device before. Before running `age`, do something like `export
TKEY_PORT=/dev/pts/22`.

## Building

First of all the device app must cloned and built in a sibling
directory. This can be done by running `git -C .. clone
https://github.com/quite/tkey-device-x25519` and follow the
instructions in the README.md over there. Then just run `make` back in
this repository.

For reproducability and maintaining a stable device app hash and thus
also a stable identity, we typically build in a container image. There
is some stuff in [contrib/](contrib/) for doing that using podman, you
can try `make` there.
