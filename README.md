Authors
=======

Andrea Barisani  
andrea.barisani@withsecure.com | andrea@inversepath.com  

Introduction
============

The GoKey application implements a USB smartcard in pure Go with support for:

  * [OpenPGP 3.4](https://gnupg.org/ftp/specs/OpenPGP-smart-card-application-3.4.pdf)
  * [FIDO U2F](https://fidoalliance.org/specs/fido-u2f-v1.2-ps-20170411/fido-u2f-overview-v1.2-ps-20170411.pdf)
  * [age plugin](https://github.com/FiloSottile/age)
  * [PKCS#11 over RPC](https://github.com/google/go-p11-kit)

In combination with the [TamaGo framework](https://github.com/usbarmory/tamago)
GoKey is meant to be executed on ARM bare metal on hardware such as the
[USB armory Mk II](https://github.com/usbarmory/usbarmory/wiki).

> :warning: the SSH management console only works on Linux or macOS hosts.

![GoKey demo](https://github.com/usbarmory/GoKey/wiki/media/gokey-usage.gif)

Security model
--------------

When running on [secure booted](https://github.com/usbarmory/usbarmory/wiki/Secure-boot-(Mk-II))
NXP i.MX6ULL devices, the built-in Data Co-Processor (DCP) is used to provide
device specific hardware encryption.

A device specific random 256-bit OTPMK key is fused in each NXP i.MX6ULL SoC at
manufacturing time, this key is unreadable and can only be used by the DCP for
AES encryption/decryption of user data, through the Secure Non-Volatile Storage
(SNVS) companion block.

The OTPMK is used to derive device specific keys, which can be used for the
following operations:

* Bundling of OpenPGP/SSH/U2F private keys, within the GoKey firmware, in
  encrypted form. This ensures that bundled keys are authenticated, confidential
  and only decrypted on a specific unit.

  The SSH private key can be uniquely and deterministically generated for each
  hardware unit, rather than being bundled in the firmware, if not passed at
  compilation time.

* Creation of the AES256 Data Object used by PSO:DEC (in AES mode) and PSO:ENC,
  this entails that AES encryption/decryption operations can only be executed
  on a specific unit.

* Generation and unwrapping of [age](https://github.com/FiloSottile/age)
  identities, AES encypted/decrypted with an OTPMK derived key.

On units which are *not* secure booted (not recommended):

* The OpenPGP private key is bundled without hardware encryption, its sole
  protection can therefore be encryption with the user passphrase (if present
  in the key).

* The SSH and U2F private keys are bundled without hardware encryption, and
  therefore readable from the firmware image.

  The SSH private key is randomly generated at each boot if not passed at
  compilation time.

* The U2F master key is derived from unique hardware serial numbers and
  the SoC unique ID, both are readable from a stolen device without secure boot
  in place.

* PSO:DEC (in AES mode) and PSO:ENC are not available.

For certain users and uses, a non secure booted device might lead to an
acceptable level of risk in case of a stolen device, nonetheless it is highly
recommended to always use a secure booted device for all configurations and to
leverage on SNVS features (see _Compiling_).

On a secure booted unit the GoKey firmware, bundled with private keys encrypted
with the device unique key, can be built by compiling on the device itself with
the [mxs-dcp kernel module](https://github.com/usbarmory/mxs-dcp)
loaded.

The module is included in all [USB armory Mk II](https://github.com/usbarmory/usbarmory/wiki)
standard [Debian base image releases](https://github.com/usbarmory/usbarmory-debian-base_image).

Deviations from OpenPGP standard support
----------------------------------------

These are security features, not bugs:

* PW3 is not implemented and card personalization is managed outside OpenPGP
  specifications to reduce the attack surface (see _Management_).

* The VERIFY command user PIN (PW1) is the passphrase of the relevant imported
  key for the requested operation (the PSO:ENC operation does not use any
  OpenPGP key, however the decryption subkey passphrase is still used for
  cardholder authentication).

* The optional key derived format (KDF) is not supported to avoid the
  transmission and internal storage of passwords in plain format, rather the
  user can issue the key passphrase over SSH for improved security
  (see _Management_).

* To prevent plaintext transmission of the PIN/passphrase, the VERIFY command
  will take any PIN (>=6 characters) if the relevant OpenPGP key has been
  already unlocked over SSH (see _Management_).

* On bare metal secure booted i.MX6ULL, the AES256 Data Object for PSO:DEC (in
  AES mode) and PSO:ENC is internally created with a device specific random
  256-bit key (OTPMK). This means that PSO:DEC and PSO:ENC can only
  decrypt/encrypt, using AES, data on the same device.

These are current limitations:

* Only signature (Sig) and decryption (Dec) keys are supported, authentication
  keys (Aut) are pending [this pull request](https://github.com/keybase/go-crypto/pull/86).

* PW1 and DSO counters are volatile (e.g. not permanent across reboots), other
  such as RC and PW3 are unused due to lack of functionality.

* Data Object 0x7f21 (Cardholder certificate) is not implemented.

Comparison with conventional smartcards
---------------------------------------

A conventional smartcard authenticates the user typically with a numeric PIN,
this cardholder authentication unlocks the code paths that allow use of key
material and/or management functions. The secret key material is stored in an
internal flash unencrypted, relaying on the physical barrier and simple
read protection mechanisms for its security.

In other words a conventional smartcard does not employ encryption of data at
rest and its code has internal access to key material unconditionally.

Futher, traditional smartcards require physical security measures to prevent
tampering attacks such as glitching or memory extraction, as there is
opportunity for an attacker in possession of a stolen card to try and extract
key material.

The GoKey firmware, when running on the
[USB armory Mk II](https://github.com/usbarmory/usbarmory/wiki),
employs a different security model as all data at rest is either authenticated,
by secure boot, or encrypted. The private OpenPGP keys are actually encrypted
twice, the first layer for the benefit of the hardware so that only
authenticated code can unwrap the key and the second layer for the benefit of
user authentication.

Therefore the GoKey firmware does not need to be stored on an internal flash
with read protection but is meant to be accessible by anyone, as it is
authenticated by the hardware and only holds encrypted content which can be
unlocked by a specific device *and* a specific user.

Additionally, to help mitigating attacks against the first layer of hardware
key wrapping, hardware decryption can be configured to take place only when a
user is successfully authenticated through the management interface.

The security model of GoKey, opposed to conventional smartcards, entails that a
stolen device gives no opportunity for an attacker to extract private key
material unless the user private SSH key (or secure boot process) as well as
OpenPGP key passphrases are compromised, when all security features are used.

Last but not least, thanks to the [TamaGo framework](https://github.com/usbarmory/tamago),
GoKey on the [USB armory Mk II](https://github.com/usbarmory/usbarmory/wiki)
employs a runtime environment written in pure high-level, memory safe, Go code
and without the dependency of an OS, or any other C based dependency. This
dramatically reduces the attack surface while increasing implementation
trustworthiness.

The following table summarizes the fundamental technological differences with a
traditional smartcard:

| Hardware type             | Trust anchor     | Data protection      | Runtime environment | Application environment | Requires tamper proofing | Encryption at rest |
|:--------------------------|------------------|----------------------|---------------------|-------------------------|--------------------------|--------------------|
| Traditional smartcards    | Flash protection | Flash protection     | JCOP                | JCOP applets            | Yes                      | No                 |
| GoKey on USB armory Mk II | Secure boot      | SoC security element | Bare metal Go       | Bare metal Go           | No                       | Yes                |

The following table summarizes the multiple authentication options available,
depending on OpenPGP and GoKey configuration, which enhance the traditional
smartcard authentication model:

| OpenPGP passphrase | GoKey authentication (see `SNVS=ssh`) | Comment                                                                                         |
|:-------------------|---------------------------------------|-------------------------------------------------------------------------------------------------|
| None               | None                                  | No security, device can be used without any authentication                                      |
| Yes over VERIFY    | None                                  | Low security, passphrase transmitted in plaintext over USB                                      |
| Yes over SSH       | None                                  | Better security, passphrase transmitted securely                                                |
| Yes over VERIFY    | Yes                                   | Good security, plaintext passphrase but standard SSH authentication required to enable key use  |
| None               | Yes                                   | Good security and convenience, standard SSH authentication required for hardware key decryption |
| Yes over SSH       | Yes                                   | High security, standard SSH authentication enables key use, passphrase transmitted securely     |

Tutorial
========

The next sections detail the compilation, configuration, execution and
operation for bare metal, virtualized and simulated environments.

A simplified tutorial for secure booted USB armory Mk II boards is available in
the [project wiki](https://github.com/usbarmory/GoKey/wiki).

Compiling
=========

Unless otherwise stated, all commands shown are relative to this repository
directory:

```
git clone https://github.com/usbarmory/GoKey && cd GoKey
```

As a pre-requisite for all compilation targets, the following environment
variables must be set or passed to the make command:

* `SNVS`: when set to a non empty value, use hardware encryption for OpenPGP,
  SSH, U2F and age private keys wrapping, see the _Security Model_ section for
  more information.

  If set to "ssh", OpenPGP and U2F key decryption, rather than executed at
  boot, must be initialized by the user over SSH (see _Management_). This improve
  resilience against physical hardware attacks as the SNVS decryption process
  cannot happen automatically on a stolen devices.

  This option can only be used when compiling on a [secure booted](https://github.com/usbarmory/usbarmory/wiki/Secure-boot-(Mk-II))
  [USB armory Mk II](https://github.com/usbarmory/usbarmory/wiki).

> :warning: the SSH management console only works on Linux or macOS hosts, it
> must be disabled for Windows hosts.

* `SSH_PUBLIC_KEY`: public key for SSH client authentication by the network
  management interface (see _Management_). If empty the SSH interface is
  disabled.

* `SSH_PRIVATE_KEY`: private key for SSH client authentication of the
  management interface SSH server (see _Management_). The key must not have a
  passphrase. When SNVS is set the key is encrypted, before being bundled, for a
  specific hardware unit.

  On secure booted units, if left empty, the SSH server key is uniquely and
  deterministically generated for each hardware unit.

  On units which are not secure booted, if left empty, the SSH server key is
  randomly generated at each boot.

OpenPGP
-------

* `PGP_SECRET_KEY`: OpenPGP secret keys in ASCII armor format, bundled
  in the output firmware. If empty OpenPGP smartcard support is disabled.

  When SNVS is set the key is encrypted, before being bundled, for a specific
  hardware unit.

* `URL`: optional public key URL.

* `NAME`, `LANGUAGE`, `SEX`: optional cardholder related data elements.

OpenPGP smartcard secret keys are typically made of 3 subkeys: signature,
decryption, authentication.

The GoKey card cannot import keys while running as any runtime change is not
allowed (see _Deviations from OpenPGP standard support_), only key bundling at
compile time is currently supported.

There are several resources on-line on OpenPGP key creation and should all be
applicable to GoKey as long as the smartcard specific `keytocard` command is
not used, but rather keys are exported armored and passed via `PGP_SECRET_KEY`
at compile time.

Some good references to start:
  * [Subkeys](https://wiki.debian.org/Subkeys)
  * [Creating newer ECC keys for GnuPG](https://www.gniibe.org/memo/software/gpg/keygen-25519.html)

Finally always ensure that existing keys are imported with minimal content, an
example preparation is the following:

```
gpg --armor --export-options export-minimal,export-clean --export-secret-key ID
```

> :warning: Please note that only RSA, ECDSA, ECDH keys are supported. Any
> other key (such as ElGamal, Ed25519) will not work.

U2F keys
--------

To enable U2F support using the [fidati](https://github.com/gsora/fidati)
library, the following variables can be set:

* `U2F_PUBLIC_KEY`: U2F device attestation certificate, if empty the U2F
  interface is disabled.

* `U2F_PRIVATE_KEY`: U2F device attestation private key, if empty the U2F
  interface is disabled.

  When SNVS is set the key is encrypted, before being bundled, for a specific
  hardware unit.

The attestation key material can be created using the
[gen-cert](https://github.com/gsora/fidati/tree/master/cmd/gen-cert) tool from
the [fidati](https://github.com/gsora/fidati) library.

On USB armory Mk II rev. β models the ATECC608B security element is used as
hardware backed monotonic counter for U2F purposes. The counter runs out at
2097151, which is considered a range sufficient for its intended purpose.

On USB armory Mk II rev. γ models a 32-bit monotonic counter is saved on the
internal eMMC in an unused reserved sector.

The U2F library performs peer-specific key derivation using a master secret
([U2F Key Wrapping](https://www.yubico.com/blog/yubicos-u2f-key-wrapping)),
GoKey derives such master secret using the SNVS to obtain an authenticated
device specific value.

When the management interface is disabled, FIDO U2F user presence is
automatically acknowledged, otherwise it can be configured at initialization
through the management interface (see _Management_).

Building the bare metal executable
----------------------------------

Build the [TamaGo compiler](https://github.com/usbarmory/tamago-go)
(or use the [latest binary release](https://github.com/usbarmory/tamago-go/releases/latest)):

```
wget https://github.com/usbarmory/tamago-go/archive/refs/tags/latest.zip
unzip latest.zip
cd tamago-go-latest/src && ./all.bash
cd ../bin && export TAMAGO=`pwd`/go
```

Please note that if performed on the USB armory Mk II, due to this
[issue](https://github.com/golang/go/issues/37122), this requires adding some
temporary [swap space](http://raspberrypimaker.com/adding-swap-to-the-raspberrypi/)
to be disabled and removed after this step is completed (to prevent eMMC wear),
alternatively you can cross compile from another host or use the
[latest binary release](https://github.com/usbarmory/tamago-go/releases/latest)).

Build the `gokey.imx` application executable with the desired variables:

```
make imx CROSS_COMPILE=arm-none-eabi- NAME="Alice" PGP_SECRET_KEY=<secret key path> SSH_PUBLIC_KEY=<public key path> SSH_PRIVATE_KEY=<private key path>
```

For signed images to be executed on [secure booted](https://github.com/usbarmory/usbarmory/wiki/Secure-boot-(Mk-II))
USB armory Mk II devices, which enable use of Secure Non-Volatile Storage
(SNVS), the `imx_signed` target should be used with the relevant `HAB_KEYS`
set:

```
make imx_signed CROSS_COMPILE=arm-none-eabi- NAME="Alice" PGP_SECRET_KEY=<secret key path> SSH_PUBLIC_KEY=<public key path> SSH_PRIVATE_KEY=<private key path> HAB_KEYS=<secure boot keys path> SNVS=ssh
```

OpenPGP host configuration
==========================

macOS
-----

The GoKey USB smartcard works out of the box on modern macOS installations with
[GPG Suite](https://gpgtools.org/).

Linux
-----

The GoKey USB smartcard has been tested with [libccid](https://ccid.apdu.fr/),
used by [OpenSC](https://github.com/OpenSC/OpenSC/wiki) on most Linux
distributions.

The libccid library [now supports](https://ccid.apdu.fr/ccid/shouldwork.html#0x12090x2702) GoKey vendor
and product IDs, if your installed version is older than this change apply the
following instructions.

To enable detection an entry must be added in `libccid_Info.plist` (typically
located in `/etc`):

* Locate the `ifdVendorID` array and add the following at its bottom:

```
<string>0x1209</string>
```

* Locate the `ifdProductID` array and add the following at its bottom:

```
<string>0x2702</string>
```

* Locate the `ifdFriendlyName` array and add the following at its bottom:

```
<string>USB armory Mk II</string>
```

The GoKey USB smartcard, once the CCID driver entries are added as in the
previous section, can then be used as any other smartcard via OpenSC on Linux.

You can refer to [Arch Linux smartcards documentation](https://wiki.archlinux.org/index.php/Smartcards)
for configuration documentation.

Windows
-------

Windows does not support Ethernet over USB devices implemented with CDC-ECM,
therefore the SSH management console must be disabled (e.g. `SNVS` can be non
empty but not to `ssh`, see _Compiling_)

Smartcard operation has not been tested but should be possible with software
that uses up-to-date smartcard drivers.

Executing
=========

USB armory Mk II: imx image
---------------------------

Follow [these instructions](https://github.com/usbarmory/usbarmory/wiki/Boot-Modes-(Mk-II)#flashing-bootable-images-on-externalinternal-media)
using the built `gokey.imx` or `gokey_signed.imx` image.

USB armory Mk II: existing bootloader
-------------------------------------

Copy the built `gokey` binary on an external microSD card (replace `$dev` with
`0`) or the internal eMMC (replace `$dev` with `1`), then launch it from the
U-Boot console as follows:

```
ext2load mmc $dev:1 0x90000000 gokey
bootelf -p 0x90000000
```

For non-interactive execution modify U-Boot configuration accordingly.

Operation
=========

GoKey can be conveniently accessed from either the USB armory Type-C plug (e.g.
USB armory directly connected to a Type-C host port) or receptacle (e.g. USB
armory connected with a Type-C to Type-A cable).

Management
----------

When running on bare metal the GoKey firmware exposes, on top of the USB CCID
smartcard and/or U2F token interfaces, an SSH server started on
[Ethernet over USB](https://github.com/usbarmory/usbarmory/wiki/Host-communication).

The SSH server authenticates the user using the public key passed at
compilation time with the `SSH_PUBLIC_KEY` environment variable. Any username
can be passed when connecting. If empty the SSH interface is disabled.

A private key for the SSH server can be optionally passed at compilation time
with the `SSH_PRIVATE_KEY` environment variable, on secure booted units it can
also be deterministically generated at each boot (see _Compiling_).

The server responds on address 10.0.0.10, with standard port 22, and can be
used to securely message passphrase verification, in alternative to smartcard
clients which issue unencrypted VERIFY commands with PIN/passphrases, signal
U2F user presence and perform additional management functions.

```
  help                          # this help
  exit, quit                    # close session
  rand                          # gather 32 bytes from TRNG via crypto/rand
  reboot                        # restart
  status                        # display smartcard/token status

  init                          # initialize OpenPGP smartcard
  lock   (all|sig|dec)          # OpenPGP key(s) lock
  unlock (all|sig|dec)          # OpenPGP key(s) unlock, prompts passphrase

  rpc                           # PKCS#11 RPC socket
                                # use with 'ssh -L p11kit.sock:127.0.0.1:22'

  age-plugin (gen|identity-v1)  # handle age plugin state machine

  u2f                           # initialize U2F token w/  user presence test
  u2f !test                     # initialize U2F token w/o user presence test
  p                             # confirm user presence
```

Note that to prevent plaintext transmission of the PIN/passphrase, the VERIFY
command requested by any OpenPGP host client will take any PIN (>= 6
characters) if the relevant OpenPGP key has been already unlocked over SSH.

OpenPGP smartcard
-----------------

You should be able to use the GoKey smartcard like any other OpenPGP card, you
can test its operation with the following commands:

* OpenSC detection: `pcsc_scan`

* OpenSC explorer: `opensc-explorer`

* OpenPGP tool key information: `openpgp-tool -K`

* GnuPG card status: `gpg --card-status` (`>` shows keys stored on a smartcard)

U2F token
---------

The U2F functionality can be used with any website or application that supports
FIDO U2F.

When the SSH interface is enabled (see _Management_) the U2F functionality must
be initialized with the `u2f` command, user presence can be demonstrated with
the `p` command (not required if `u2f !test` is used for initialization).

When the SSH interface is disabled user presence is automatically acknowledged
at each request.

age plugin
----------

The [age plugin](https://github.com/FiloSottile/age) functionality is available
only on secure booted units (e.g. `SNVS` set to non empty value when
_Compiling_).

An age key pair can be generated through the _Management_ interface with the
`age gen` command:

```
ssh 10.0.0.1 age-plugin gen

# public key: age1u7q3elfae7wawlneek660ayqd8270u6c35mdt246rmq79z8k5p2qqwxry6
AGE-PLUGIN-GOKEY-1EPZFX6TNDUCF55FQLU83VW820GZGKMU2QSZ577SCH8CLKXGHXS7JZLPRY8...
```

The generated recipient and identity file can now be used with an age plugin
that relays the `identity-v1` command, over SSH, to the GoKey _Management_
interface.

The following example shell wrapper can be used by an age client if present in
the `PATH`:

```shell
#!/bin/sh

OPTS=$(getopt -n age-plugin-gokey --options p: --longoptions 'age-plugin:' -- $@)

eval set -- "$OPTS"

while (($#)); do
	case $1 in
		-p|--age-plugin)	sm=$2; shift;;
		--)			shift; break;;
		*)			echo "invalid argument";;
	esac
	shift
done

ssh 10.0.0.1 age-plugin $sm
```

Once the plugin wrapper is present in the `PATH`, the GoKey generated identity
can be used as follows while the GoKey device is reachable via its _Management_
interface:

```
age -a -R gokey.recipient -o secret.txt.age -e secret.txt
age -i gokey.identity -d secret.txt.age
```

PKCS#11 token
-------------

When the SSH interface is enabled (see _Management_) a GoKey protected
ECDSA/RSA key can be exposed through
[PKCS#11 over RPC](https://github.com/google/go-p11-kit) through an
[SSH forwarded Unix socket](https://p11-glue.github.io/p11-glue/p11-kit/manual/remoting.html).

A direct SSH TCP forward to a Unix socket exposes GoKey PKCS#11 over RPC
interface:

```
$ ssh -N -L p11kit.sock:127.0.0.1:22 10.0.0.10
```

This allows use of such key through PKCS#11 API clients, the following example
illustrates integration with OpenSSL.

```
$ export P11_KIT_SERVER_ADDRESS=unix:path=p11kit.sock
$ pkcs11-tool --module /usr/lib/pkcs11/p11-kit-client.so --list-slots

Available slots:
Slot 0 (0x1): example-slot
  token label        : GoKey
  token manufacturer : WithSecure Foundry
  token model        : USB armory Mk II
  token flags        : token initialized, readonly
  hardware version   : 2.0
  firmware version   : 0.1
  serial num         : C148261A
  pin min/max        : 0/0

$ pkcs11-tool --module /usr/lib/pkcs11/p11-kit-client.so --list-objects

Using slot 0 with a present token (0x1)
Certificate Object; type = X.509 cert
  subject:    DN:
  serial:     75A3F93B15D62D15
  ID:         010275a3f93b15d62d15
Private Key Object; RSA
  Usage:      decrypt, sign
  Access:     sensitive, always sensitive, never extractable
Public Key Object; RSA 2048 bits
  ID:         010275a3f93b15d62d15
  Usage:      encrypt, verify
  Access:     none
```

LED status
----------

On the [USB armory Mk II](https://github.com/usbarmory/usbarmory/wiki)
the LEDs are used as follows:

| LED           | On                                               | Off                                    |
|:-------------:|--------------------------------------------------|----------------------------------------|
| blue + white  | at startup: card is initializing¹                | card has been initialized              |
| blue          | one or more OpenPGP private subkeys are unlocked | all OpenPGP private subkeys are locked |
| white         | OpenPGP security operation in progress           | no security operation in progress      |
| white         | blinking: U2F user presence is requested         | no presence requested                  |

¹ With `SNVS=ssh` both LEDs remain on until the `init` command has been issued over SSH management interface.

Debugging
=========

Virtual Smart Card
------------------

The [virtual smart card project](http://frankmorgner.github.io/vsmartcard/virtualsmartcard/README.html)
allows testing of GoKey OpenPGP functionality in userspace.

Build the `gokey_vpcd` application executable:

```
make gokey_vpcd PGP_SECRET_KEY=<secret key path>
```

On the host install [vsmartcard](http://frankmorgner.github.io/vsmartcard/index.html),
Arch Linux users can use the [virtualsmartcard AUR package](https://aur.archlinux.org/packages/virtualsmartcard/).

Ensure that a configuration for `vpcd` is added in your `pcsc` configuration.
On Arch Linux the following is automatically created in `/etc/reader.conf.d`:

```
FRIENDLYNAME "Virtual PCD"
DEVICENAME   /dev/null:0x8C7B
LIBPATH      //usr/lib/pcsc/drivers/serial/libifdvpcd.so
CHANNELID    0x8C7B
```

Launch the PC/SC daemon:

```
sudo systemctl start pcscd
```

Launch the built `gokey_vpcd` executable:

```
./gokey_vpcd -c 127.0.0.1:35963
```

The same executable can also be used to test the _PKCS#11 token_ interface, a
relevant `P11_KIT_SERVER_ADDRESS` variable is returned upon execution of
`gokey_vpcd`.

Send manual commands to GnuPG smart-card daemon (SCD)
-----------------------------------------------------

* Example for issuing a GET CHALLENGE request to get 256 random bytes (due to
  SCD protocol formatting some post processing required to extract the actual
  random output):

```
gpg-connect-agent "SCD RANDOM 256" /bye | perl -pe 'chomp;s/^D\s//;s/%(0[AD]|25)/chr(hex($1))/eg;if(eof&&/^OK$/){exit}'
```

License
=======

GoKey | https://github.com/usbarmory/GoKey  
Copyright (c) WithSecure Corporation

This program is free software: you can redistribute it and/or modify it under
the terms of the GNU General Public License as published by the Free Software
Foundation under version 3 of the License.

This program is distributed in the hope that it will be useful, but WITHOUT ANY
WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
PARTICULAR PURPOSE. See the GNU General Public License for more details.

See accompanying LICENSE file for full details.
