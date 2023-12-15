
Plugin for [age](https://github.com/FiloSottile/age) to use a Tillitis
[TKey](https://github.com/tillitis/tillitis-key1) USB security key.

For doing X25519 ECDH it embeds and runs the device app
[tkey-device-x25519](https://github.com/quite/tkey-device-x25519) on
the TKey. The Go package
[tkeyx25519](https://github.com/quite/tkeyx25519) is used for
communicating with this device app.

Note that this should still be considered is work in progress (WIP).
In particular, there is a possibility that we could need to make
changes to tkey-device-x25519 for some reason. Changes to the source
code would change the binary, which would cause the identity of a TKey
that runs it to be different. This would mean that the secret key no
longer is the same, and that data encrypted for an identity based on
the previous secret would be impossible to decrypt. It could be
possible to work around this by first decrypting using an older
version, and then encrypting again using a later.

## Installing

The easiest way to install is to run:

```
go install github.com/quite/age-plugin-tkey/cmd/age-plugin-tkey@latest
```

Se below for information about building yourself.

If you have not installed and used any other software for the TKey
before, you might not be able to access the serial port of the TKey.
One way to solve that is by executing the following as root:

```
cp -a 60-tkey.rules /etc/udev/rules.d/
udevadm control --reload
udevadm trigger
```

You could also add yourself to the group that owns `/dev/ttyACM0` on
your system.

## Using

In the following we create a new keypair/identity and learn about the
public key/recipient that is us. Then we encrypt a note to ourselves,
and proceed to decrypt it. The LED on the TKey will shine yellow when
the X25519 app has been loaded (and will flash in the same colour when
it needs to be touched).

```
$ age-plugin-tkey --generate >my-keys
# recipient: age1xuqv8tq5ttkgwe3quys0dfwxv6zzqpemvckjeutudtjjhfac2f9q6lc377
$ echo "remember to fix all bugs!" | age --encrypt -a -r age1xuqv8tq5ttkgwe3quys0dfwxv6zzqpemvckjeutudtjjhfac2f9q6lc377 >note-to-self
$ age -i my-keys --decrypt ./note-to-self
remember to fix all bugs!
```

The generated identity will by default cause TKey to require physical
touch before computing a shared key (doing ECDH). You can pass the
flag `--no-touch` to generate an identity that does not.

After running the above, te file `my-keys` ends up containing a line
beginning with `AGE-PLUGIN-TKEY-`. This holds the parameters used for
deriving the secret key on the TKey (which must be the exact same
physical device as used for generation). The secret key itself never
leaves the TKey hardware, which actually has no storage. You can learn
more about this here:
https://dev.tillitis.se/intro/#measured-boot--secrets

To run the plugin towards an emulated TKey in QEMU, instead of real
hardware, you can set the environment variable `AGE_TKEY_PORT` to
QEMU's char-device (shown when QEMU starts up). Before running `age`,
do something like `export AGE_TKEY_PORT=/dev/pts/22`.

## Building

For ease of installing and building the age-plugin-tkey Go program, we
keep the device app binary committed to the repository. So if you
don't want or need to rebuild or work on the device app, you can just
build with:

```
make
```

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

The file `internal/tkey/x25519-hashes.sha512` contains hashes of known
versions of the device app binary, as built using our container image
(which currently has clang 17). We keep the device app binaries
committed to the repository. A tagged version of age-plugin-tkey
currently uses one specific version of that binary, which is embedded
during build, see [internal/tkey/tkey.go](internal/tkey/tkey.go).
