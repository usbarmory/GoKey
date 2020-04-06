// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"bytes"
	"math"

	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/armor"
	"github.com/keybase/go-crypto/openpgp/packet"
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

		// https://github.com/golang/go/issues/25983
		//if subkey.Sig.FlagAuthenticate {
		//	aut = &subkey
		//}
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

// p19, 5.4.1. Uncompressed Point Format for NIST Curves, RFC8422
func getUncompressedPointFormat(x []byte, y []byte, octets int) []byte {
	buf := new(bytes.Buffer)

	// uncompressed(4)
	buf.WriteByte(0x04)

	for _, v := range [][]byte{x, y} {
		// padding
		for i := int(math.Ceil(float64(octets) / 8)); i > len(v); i-- {
			buf.WriteByte(0x00)
		}

		buf.Write(v)
	}

	return buf.Bytes()
}
