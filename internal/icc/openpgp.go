// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"bytes"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

func decodeArmoredKey(key []byte) (entity *openpgp.Entity, err error) {
	k := bytes.NewBuffer(key)
	keyBlock, err := armor.Decode(k)

	if err != nil {
		return
	}

	reader := packet.NewReader(keyBlock.Body)
	entity, err = openpgp.ReadEntity(reader)

	return
}

func decodeSubkeys(entity *openpgp.Entity) (sig *openpgp.Subkey, dec *openpgp.Subkey, aut *openpgp.Subkey) {
	for i, subkey := range entity.Subkeys {
		if subkey.Sig == nil || !subkey.Sig.FlagsValid {
			continue
		}

		if subkey.Sig.FlagSign {
			sig = &entity.Subkeys[i]
		}

		if subkey.Sig.FlagEncryptStorage && subkey.Sig.FlagEncryptCommunications {
			dec = &entity.Subkeys[i]
		}

		if subkey.Sig.FlagAuthenticate {
			aut = &subkey
		}
	}

	return
}

func getOID(name string) (oid []byte) {
	// p99, 10 Domain parameter of supported elliptic curves, OpenPGP application Version 3.4
	switch name {
	case "P-256":
		oid = []byte{0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07}
	case "P-384":
		oid = []byte{0x2B, 0x81, 0x04, 0x00, 0x22}
	case "P-521":
		oid = []byte{0x2B, 0x81, 0x04, 0x00, 0x23}
	case "brainpoolP256r1":
		oid = []byte{0x2B, 0x24, 0x03, 0x03, 0x02, 0x08, 0x01, 0x01, 0x07}
	case "brainpoolP384r1":
		oid = []byte{0x2B, 0x24, 0x03, 0x03, 0x02, 0x08, 0x01, 0x01, 0x0B}
	case "brainpoolP512r1":
		oid = []byte{0x2B, 0x24, 0x03, 0x03, 0x02, 0x08, 0x01, 0x01, 0x0D}
	default:
		oid = []byte{0x2B, 0x81, 0x04, 0x00, 0x23}
	}

	return
}
