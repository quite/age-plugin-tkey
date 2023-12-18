[age](https://github.com/FiloSottile/age) plugin for using a Tillitis
[TKey](https://github.com/tillitis/tillitis-key1) USB security key.

For X25519 ECDH key agreement it embeds and runs the
[tkey-device-x25519](https://github.com/quite/tkey-device-x25519) app
on the TKey. The Go package
[tkeyx25519](https://github.com/quite/tkeyx25519) is used for
communicating with the device app.

Note that this should still be considered work in progress (WIP). In
particular, there is a possibility that we could need to make changes
to tkey-device-x25519 for some reason. That said, I think current
design and its implementation may be sound.

Changes to the source code of tkey-device-x25519 would change the
binary, which would cause the identity of a TKey that runs it to be
different. This would mean that the secret key no longer is the same,
and that data encrypted for an identity based on the previous secret
would be impossible to decrypt. It could be possible to work around
this by first decrypting using an older version, and then encrypting
again using a later.

## Installing

The easiest way is to run the following, which installs the
`age-plugin-tkey` binary in `$GOPATH/bin` (or `$HOME/go/bin` if GOPATH
is not set).

```
go install github.com/quite/age-plugin-tkey/cmd/age-plugin-tkey@latest
```

`age-plugin-tkey` needs to be installed in a directory listed in
`PATH`, so that `age` can find it.

See below for information about building yourself.

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

In the following we create a new random *identity* (private key) for
ourselves and learn about the corresponding *recipient* (public key).
Then we encrypt a message to ourselves using that recipient, and
proceed to decrypt it using our identity. The LED on the TKey will
shine yellow when the X25519 app has been loaded, and will flash in
the same colour when it needs to be touched (upon decryption).

```
$ age-plugin-tkey --generate --output my-identity
# recipient: age1xuqv8tq5ttkgwe3quys0dfwxv6zzqpemvckjeutudtjjhfac2f9q6lc377
$ echo "remember to fix all bugs!" | age --encrypt -a -r age1xuqv8tq5ttkgwe3quys0dfwxv6zzqpemvckjeutudtjjhfac2f9q6lc377 >note-to-self
$ age -i my-identity --decrypt ./note-to-self
remember to fix all bugs!
```

After running this, the file `my-identity` will contain a line
beginning with `AGE-PLUGIN-TKEY-` which holds parameters that were
used on the TKey when generating the identity, and will be used again
to recreate it before computing a shared key (to be used for
decryption of the message). The file `my-identity` file should be
treated as a secret, even though the full identity is a combination of
data in this file and key material on the TKey -- which is unique to
each TKey, and never leaves the hardware. There is actually no storage
on the TKey, you can learn more about how it derives secret keys here:
https://dev.tillitis.se/intro/#measured-boot--secrets

The file also has some comment lines beginning with `#` with more
information about the identity. Especially useful is the `recipient`,
which is used to encrypt data which then can be decrypted this precise
identity.

You can generate as many identities as you want, each will be bound to
the unique TKey it was generated with, and have its own corresponding
recipient. The *recipient* is not a secret, on the contrary it is what
you give to your friend so they can encrypt their message for you. The
corresponding identity and the same physical TKey device are both
needed in order to decrypt it. The `age` option `-i/--identity` takes
a file, which can contain one or more identity lines.

The generated identity will by default cause TKey to require physical
touch before computing a shared key (doing ECDH). You can pass the
flag `--no-touch` to generate an identity that does not.

To run the plugin towards an emulated TKey in QEMU instead of real
hardware, you can set the environment variable `AGE_TKEY_PORT` to
QEMU's char-device (shown when QEMU starts up). Before running `age`,
do something like `export AGE_TKEY_PORT=/dev/pts/22`.

### Multiple recipients

Files encrypted for multiple recipients are indeed handled by this
plugin. There is however one gotcha which might be slightly confusing
and annoying at times. If your TKey identity requires touch (which
they do by default), then you may have to touch your TKey more than
one time when decrypting a single piece of data that has been
encrypted for more than one recipient.

The best you can do to avoid this when encrypting for multiple
recipients using `age` is to pass any TKey recipient with the *first*
`-r`. Also, when decrypting, if the identity file you're passing with
`-i` has multiple identities, then make sure any TKey identity line
comes last. This will avoid "unnecessary" touching if the data can be
decrypted using an earlier non-TKey identity. Below follows a more
in-depth explanation.

Recipients for identities based in TKey have the same format as
recipients for identities generated by the native `age-keygen`. This
ought to be because TKey also use native X25519 keys.

The consequence is that when you are decrypting using a TKey identity
(`-i`) and the data is encrypted for multiple recipients, then an
attempt to use the TKey (for ECDH) will be made for each native X25519
recipient, in the order they were passed with `-r` -- but I guess
implementations could vary. Imagine some data encrypted and then
decrypted like this:

```
age-keygen -o some-agekeygen-id
# Note the RECIPIENT (public key) for this some-agekeygen-id

age-plugin-tkey --generate -o my-tkey-id
# Note the RECIPIENT (public key) for this my-tkey-id

printf "some data\n" | age -r SOME-AGEKEYGEN-RECIPIENT -r MY-TKEY-RECIPIENT --encrypt -o ./ciphertext

age -i my-tkey-id --decrypt ./ciphertext
```

When decrypting, two recipients are found to be native X25519, so both
will have to be tried using the TKey identity (`-i`). The first
recipient (non-TKey) passed using `-r` will be tried first. You will
have to touch your TKey, but decryption will fail silently. Then the
second recipient (TKey) will be tried. You will have to touch TKey
again, but this time decryption will succeed.

## Building

For ease of installing and building the `age-plugin-tkey` Go program, we
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
runs [build.sh](build.sh) there). `podman` is used for running the
container (packages: `podman rootlesskit slirp4netns`).

The file
[internal/tkey/x25519-hashes.sha512](internal/tkey/x25519-hashes.sha512)
contains hashes of known versions of the device app binary, as built
using our container image (which currently has clang 17). We keep the
device app binaries committed to the repository. A tagged version of
age-plugin-tkey currently uses one specific version of that binary,
which is embedded during build, see
[internal/tkey/tkey.go](internal/tkey/tkey.go).
