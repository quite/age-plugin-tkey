
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

By default the generated identity will require physical touch of TKey
when it does ECDH key exchange. Use the flag `--no-touch` to generate
an identity that does not.

After running the above, te file `my-keys` ends up containing a line
beginning with `AGE-PLUGIN-TKEY-`. This holds the parameters used for
deriving the secret key on the TKey (which must be the exact same
physical device as used for generation). The secret key itself never
leaves the TKey hardware, which actually has no storage. You can learn
more about this here:
https://dev.tillitis.se/intro/#measured-boot--secrets

To run the plugin towards an emulated TKey in QEMU, instead of real
hardware, you can set the environment variable `AGE_TKEY_PORT` to
QEMU's char-device before. Before running `age`, do something like
`export AGE_TKEY_PORT=/dev/pts/22`.

## Building

For reproducibility the X25519 device app is typically built in a
container, thus locking down the toolchain, and using specific
versions of dependencies. Because if one single bit changes in the
app.bin that will run on the TKey (for example due to a newer
clang/llvm), then the identity (private/public key) of it will change.

You can use [build-in-container.sh](build-in-container.sh) to build
both the device app and age-plugin-tkey using our own container image
(see [Containerfile](Containerfile)). The clone of this repo that
you're sitting in will be mounted into the container and built, but
dependencies will be freshly cloned as they don't exist inside (it
runs `build.sh` there). `podman` is used for running the container
(packages: `podman rootlesskit slirp4netns`).

The `x25519/app.bin.sha512` contains the expected hash of the device
app binary when built using our container image which currently has
clang 17.
