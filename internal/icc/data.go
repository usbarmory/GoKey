// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/hsanjuan/go-nfctype4/apdu"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/ecdh"
)

const (
	// p22, 4.4.1 DOs for GET DATA, OpenPGP application Version 3.4
	DO_APPLICATION_IDENTIFIER      = 0x4f
	DO_LOGIN_DATA                  = 0x5e
	DO_URL                         = 0x5f50
	DO_HISTORICAL_BYTES            = 0x5f52
	DO_CARDHOLDER_RELATED_DATA     = 0x65
	DO_APPLICATION_RELATED_DATA    = 0x6e
	DO_SECURITY_SUPPORT_TEMPLATE   = 0x7a
	DO_CARDHOLDER_CERTIFICATE      = 0x7f21 // TODO
	DO_EXTENDED_LENGTH_INFORMATION = 0x7f66
	DO_PW_STATUS_BYTES             = 0xc4
	DO_KEY_INFORMATION             = 0xde
	DO_ALGORITHM_INFORMATION       = 0xfa

	// DOs not directly accessible
	DO_NAME                       = 0x5b
	DO_LANGUAGE                   = 0x5f2d
	DO_SEX                        = 0x5f35
	DO_DISCRETIONARY_DATA_OBJECTS = 0x73
	DO_EXTENDED_CAPABILITIES      = 0xc0
	DO_ALGORITHM_ATTRIBUTES_SIG   = 0xc1
	DO_ALGORITHM_ATTRIBUTES_DEC   = 0xc2
	DO_ALGORITHM_ATTRIBUTES_AUT   = 0xc3
	DO_FINGERPRINTS               = 0xc5
	DO_CA_FINGERPRINTS            = 0xc6
	DO_GENERATION_EPOCHS          = 0xcd
	DO_DIGITAL_SIGNATURE_COUNTER  = 0x93

	// p65, 7.2.11 PSO: DECIPHER, OpenPGP application Version 3.4
	DO_CIPHER = 0xa6
	DO_AES256 = 0xd5
	// p72, 7.2.14 GENERATE ASYMMETRIC KEY PAIR, OpenPGP application Version 3.4
	DO_PUB_KEY     = 0x7f49
	DO_RSA_MOD     = 0x81
	DO_RSA_EXP     = 0x82
	DO_EXT_PUB_KEY = 0x86

	// p33, 4.4.3.9 Algorithm Attributes, OpenPGP application Version 3.4
	RSA                    = 0x01
	RSA_KEY_SIZE           = 4096
	RSA_EXPONENT_SIZE      = 32
	IMPORT_FORMAT_STANDARD = 0x00

	// p33, 4.4.3.8 Key Information, OpenPGP application Version 3.4
	KEY_SIG         = 0x01
	KEY_DEC         = 0x02
	KEY_AUT         = 0x03
	KEY_NOT_PRESENT = 0x00
	KEY_GENERATED   = 0x01
	KEY_IMPORTED    = 0x02

	PW1_MAX_LENGTH = 127
	RC_MAX_LENGTH  = 127
	PW3_MAX_LENGTH = 127
)

var ATR []byte
var HISTORICAL_BYTES []byte
var EXTENDED_CAPABILITIES []byte
var EXTENDED_LENGTH []byte

// p15, 4.2.1 Application Identifier (AID), OpenPGP application Version 3.4
var RID = []byte{0xd2, 0x76, 0x00, 0x01, 0x24, 0x01}

func init() {
	var tck byte

	// Initialize ATR according to ISO/IEC 7816-4,
	// https://en.wikipedia.org/wiki/Answer_to_reset.
	ATR = []byte{
		// TS - Direct Convention
		0x3b,
		// T0 - Y(1): b1101, K: 10 (historical bytes)
		0xda,
		// TA(1) - Fi=372, Di=1, 372 cycles/ETU (10752 bits/s at 4.00 MHz, 13440 bits/s for fMax=5 MHz)
		0x11,
		// TC(1) - Extra guard time: 255
		0xff,
		// TD(1) - Y(i+1) = b1000, Protocol T=1
		0x81,
		// TD(2) - Y(i+1) = b1011, Protocol T=1
		0xb1,
		// TA(3) - IFSC: 254
		0xfe,
		// TB(3) - Block Waiting Integer: 5 - Character Waiting Integer: 5
		0x55,
		// TD(3) Y(i+1) = b0001, Protocol T=15
		0x1f,
		// TA(4) - Clock stop: not supported - Class accepted by the card: (3G) A 5V B 3V
		0x03,
		// Historical bytes
	}

	// p44, 6 Historical Byte, OpenPGP application Version 3.4
	HISTORICAL_BYTES = []byte{
		// Category indicator
		0x00,
		// Tag: 3, Len: 1 (card service data byte)
		0x31,
		//   Card service data byte: 132
		//     - Application selection: by full DF name
		//     - EF.DIR and EF.ATR access services: by GET DATA command
		//     - Card with MF
		0x84,
		// Tag: 7, Len: 3 (card capabilities)
		0x73,
		// Selection methods: 128
		//   - DF selection by full DF name
		0x80,
		// Data coding byte: 1
		//   - Behaviour of write functions: one-time write
		//   - Value 'FF' for the first byte of BER-TLV tag fields: valid
		//   - Data unit in quartets: 1
		0x01,
		// Command chaining, length fields and logical channels: 128
		//   - Command chaining
		//   - Logical channel number assignment: No logical channel
		//   - Maximum number of logical channels: 1
		0x40,
		// Mandatory status indicator (3 last bytes)
		//   LCS (life card cycle): 0 (No information given)
		//   SW: 90 00 ()
		0x00,
		0x90,
		0x00,
	}

	ATR = append(ATR, HISTORICAL_BYTES...)

	// compute ATR checksum
	for _, b := range ATR[1:] {
		tck = tck ^ b
	}

	// TCK - Checksum
	ATR = append(ATR, tck)

	// p32, 4.4.3.7 Extended Capabilities, OpenPGP application Version 3.4
	EXTENDED_CAPABILITIES = []byte{
		// support GET CHALLENGE, PSO:DEC/ENC with AES
		0x42,
		// no support for Secure Messaging
		0x00,
		// maximum length of GET CHALLENGE
		0xff, 0xff,
		// maximum length of Cardholder Certificate
		0xff, 0xff,
		// maximum length of special DOs
		0xff, 0xff,
		// PIN block 2 format not supported
		0x00,
		// MSE command for key numbers 2 (DEC) and 3 (AUT) not supported
		0x00,
	}

	// p14, 4.1.3.1 Extended length information, OpenPGP application Version 3.4
	EXTENDED_LENGTH = []byte{
		// Maximum number of bytes in a command APDU
		0xff, 0xff,
		// Maximum number of bytes in a response APDU
		0xff, 0xff,
	}
}

// AID implements
// p15, 4.2.1 Application Identifier (AID), OpenPGP application Version 3.4.
func (card *Interface) AID() (aid []byte) {
	aid = RID
	// version (3.4)
	aid = append(aid, []byte{0x03, 0x04}...)
	// manufacturer - 0xF5EC, assigned by GnuPG e.V. on 2020-02-21 to F-Secure
	aid = append(aid, []byte{0xF5, 0xEC}...)
	// serial number - place holder
	aid = append(aid, card.Serial[:]...)
	// RFU
	aid = append(aid, []byte{0x00, 0x00}...)

	return
}

// ATR returns the Answer to reset (ATR) according to ISO/IEC 7816-4.
func (card *Interface) ATR() (atr []byte) {
	return ATR
}

// CardholderRelatedData builds and returns Data Object 0x65.
func (card *Interface) CardholderRelatedData() []byte {
	data := new(bytes.Buffer)

	data.Write(tlv(DO_NAME, []byte(card.Name)))
	data.Write(tlv(DO_LANGUAGE, []byte(card.Language)))
	data.Write(tlv(DO_SEX, []byte(card.Sex)))

	return data.Bytes()
}

// DiscretionaryData builds and returns Data Object 0x73.
func (card *Interface) DiscretionaryData() []byte {
	data := new(bytes.Buffer)

	data.Write(tlv(DO_EXTENDED_CAPABILITIES, EXTENDED_CAPABILITIES))
	data.Write(tlv(DO_ALGORITHM_ATTRIBUTES_SIG, card.AlgorithmAttributes(card.Sig)))
	data.Write(tlv(DO_ALGORITHM_ATTRIBUTES_DEC, card.AlgorithmAttributes(card.Dec)))
	data.Write(tlv(DO_ALGORITHM_ATTRIBUTES_AUT, card.AlgorithmAttributes(card.Aut)))
	data.Write(tlv(DO_PW_STATUS_BYTES, card.PWStatusBytes()))
	data.Write(tlv(DO_FINGERPRINTS, card.Fingerprints()))
	data.Write(tlv(DO_CA_FINGERPRINTS, card.CAFingerprints()))
	data.Write(tlv(DO_GENERATION_EPOCHS, card.GenerationEpochs()))

	return data.Bytes()
}

// AlgorithmAttributes builds and returns the Data Objects specified at
// p34, 4.4.3.9 Algorithm Attributes, OpenPGP application Version 3.4.
func (card *Interface) AlgorithmAttributes(subkey *openpgp.Subkey) (data []byte) {
	data = make([]byte, 6)
	data[0] = RSA
	data[5] = IMPORT_FORMAT_STANDARD

	if subkey == nil || subkey.PublicKey == nil {
		binary.BigEndian.PutUint16(data[1:], uint16(RSA_KEY_SIZE))
		binary.BigEndian.PutUint16(data[3:], uint16(RSA_EXPONENT_SIZE))
		return
	}

	switch pubKey := subkey.PublicKey.PublicKey.(type) {
	case *rsa.PublicKey:
		binary.BigEndian.PutUint16(data[1:], uint16(pubKey.N.BitLen()))
		binary.BigEndian.PutUint16(data[3:], uint16(RSA_EXPONENT_SIZE))
	case *ecdsa.PublicKey:
		data = []byte{byte(subkey.PublicKey.PubKeyAlgo)}
		data = append(data, getOID(pubKey.Params().Name)...)
		data = append(data, IMPORT_FORMAT_STANDARD)
	case *ecdh.PublicKey:
		data = []byte{byte(subkey.PublicKey.PubKeyAlgo)}
		data = append(data, getOID(pubKey.Params().Name)...)
		data = append(data, IMPORT_FORMAT_STANDARD)
	default:
		log.Printf("unexpected public key type in DO_ALGORITHM_ATTRIBUTES %T", pubKey)
	}

	return data
}

// PWStatusBytes builds and returns Data Object 0xC4.
func (card *Interface) PWStatusBytes() []byte {
	status := new(bytes.Buffer)

	// PW1 only valid for one PSO:CDS command
	status.WriteByte(PW1_CDS_MULTI)
	// max. length of PW1 (user), UTF-8 or derived password
	status.WriteByte((PW1_MAX_LENGTH<<1)&0b11111110 | 0b0)
	// max. length of Resetting Code (RC) for PW1
	status.WriteByte(RC_MAX_LENGTH)
	// max. length of PW3 (admin)
	status.WriteByte(PW3_MAX_LENGTH)

	// error counter for PW1
	status.WriteByte(card.errorCounterPW1)
	// error counter for RC
	status.WriteByte(card.errorCounterRC)
	// error counter for PW3
	status.WriteByte(card.errorCounterPW3)

	return status.Bytes()
}

// Fingerprints collects card OpenPGP subkey fingerprints and returns them in
// Data Object 0xC5.
func (card *Interface) Fingerprints() (fingerprints []byte) {
	fingerprints = make([]byte, 60)
	subkeys := []*openpgp.Subkey{card.Sig, card.Dec, card.Aut}

	for i, subkey := range subkeys {
		if subkey == nil {
			continue
		}

		copy(fingerprints[i*20:], subkey.PublicKey.Fingerprint[:])
	}

	return
}

// CAFingerprints collects card OpenPGP CA fingerprints and returns them in
// Data Object 0xC6. Currently unused (always empty).
func (card *Interface) CAFingerprints() (fingerprints []byte) {
	fingerprints = make([]byte, 60)

	for i, ca := range card.CA {
		if ca == nil {
			continue
		}

		copy(fingerprints[i*20:], ca.PrimaryKey.Fingerprint[:])
	}

	return
}

// GenerationEpochs collects card OpenPGP creation times and returns them in
// Data Object 0xCD.
func (card *Interface) GenerationEpochs() (epochs []byte) {
	epochs = make([]byte, 12)

	if card.Sig != nil {
		binary.BigEndian.PutUint32(epochs, uint32(card.Sig.PublicKey.CreationTime.Unix()))
	}

	if card.Dec != nil {
		binary.BigEndian.PutUint32(epochs[4:], uint32(card.Dec.PublicKey.CreationTime.Unix()))
	}

	if card.Aut != nil {
		binary.BigEndian.PutUint32(epochs[8:], uint32(card.Aut.PublicKey.CreationTime.Unix()))
	}

	return
}

// KeyInformation implements
// p33, 4.4.3.8 Key Information, OpenPGP application Version 3.4.
//
// This information is required for Yubico OpenPGP attestation, this
// implementation doesn't (yet) support this feature as its usefulness is
// questionable. Therefore we just flag all keys as imported (which also
// happens to be the only allowed mechanism for now).
func (card *Interface) KeyInformation() []byte {
	return []byte{
		KEY_SIG, KEY_IMPORTED,
		KEY_DEC, KEY_IMPORTED,
		KEY_AUT, KEY_IMPORTED,
	}
}

// AlgorithmInformation implements
// p37, 4.4.3.11 Algorithm Information, OpenPGP application Version 3.4.
//
// The standard is ambiguous on whether this DO needs to be present if
// algorithm attributes cannot be changed, the DO table suggests this
// is mandatory but the DO description suggests otherwise.
//
// Given that this implementation does not allow changes, the imported key
// attributes are returned.
func (card *Interface) AlgorithmInformation() []byte {
	data := new(bytes.Buffer)

	data.Write(tlv(DO_ALGORITHM_ATTRIBUTES_SIG, card.AlgorithmAttributes(card.Sig)))
	data.Write(tlv(DO_ALGORITHM_ATTRIBUTES_DEC, card.AlgorithmAttributes(card.Dec)))
	data.Write(tlv(DO_ALGORITHM_ATTRIBUTES_AUT, card.AlgorithmAttributes(card.Aut)))

	return data.Bytes()
}

func (card *Interface) DigitalSignatureCounter() []byte {
	if card.digitalSignatureCounter > 0xffffff {
		card.digitalSignatureCounter = 0
	}

	counter := make([]byte, 4)
	binary.BigEndian.PutUint32(counter, card.digitalSignatureCounter)

	return counter[1:4]
}

// SecuritySupportTemplate builds and returns Data Object 0x7A.
func (card *Interface) SecuritySupportTemplate() []byte {
	return tlv(DO_DIGITAL_SIGNATURE_COUNTER, card.DigitalSignatureCounter())
}

// ApplicationRelatedData implements
// p30, 4.4.3.1 Application Related Data, OpenPGP application Version 3.4.
func (card *Interface) ApplicationRelatedData() []byte {
	data := new(bytes.Buffer)

	data.Write(tlv(DO_APPLICATION_IDENTIFIER, card.AID()))
	data.Write(tlv(DO_HISTORICAL_BYTES, HISTORICAL_BYTES))
	data.Write(tlv(DO_EXTENDED_LENGTH_INFORMATION, EXTENDED_LENGTH))
	data.Write(tlv(DO_DISCRETIONARY_DATA_OBJECTS, card.DiscretionaryData()))

	return data.Bytes()
}

// GetData implements
// p57, 7.2.6 GET DATA, OpenPGP application Version 3.4.
func (card *Interface) GetData(tag uint16) (rapdu *apdu.RAPDU, err error) {
	rapdu = CommandCompleted(nil)

	// p22, 4.4.1 DOs for GET DATA, OpenPGP application Version 3.4
	switch tag {
	case DO_APPLICATION_IDENTIFIER:
		rapdu.ResponseBody = card.AID()
	case DO_LOGIN_DATA:
		rapdu.ResponseBody = card.LoginData
	case DO_URL:
		rapdu.ResponseBody = []byte(card.URL)
	case DO_HISTORICAL_BYTES:
		rapdu.ResponseBody = HISTORICAL_BYTES
	case DO_CARDHOLDER_RELATED_DATA:
		rapdu.ResponseBody = card.CardholderRelatedData()
	case DO_APPLICATION_RELATED_DATA:
		rapdu.ResponseBody = card.ApplicationRelatedData()
	case DO_SECURITY_SUPPORT_TEMPLATE:
		rapdu.ResponseBody = card.SecuritySupportTemplate()
	case DO_EXTENDED_LENGTH_INFORMATION:
		rapdu.ResponseBody = EXTENDED_LENGTH
	case DO_PW_STATUS_BYTES:
		rapdu.ResponseBody = card.PWStatusBytes()
	case DO_KEY_INFORMATION:
		rapdu.ResponseBody = card.KeyInformation()
	case DO_ALGORITHM_INFORMATION:
		rapdu.ResponseBody = card.AlgorithmInformation()
	default:
		rapdu.SW1 = 0x6a
		rapdu.SW1 = 0x88
		log.Printf("unsupported DO tag %x", tag)
	}

	return
}

// PutData implements
// p60, 7.2.8 PUT DATA, OpenPGP application Version 3.4.
//
// This is not implemented (always returns command not allowed) as card
// personalization is managed outside OpenPGP specifications and the PW3 PIN
// (required for this command) is not supported.
func (card *Interface) PutData(tag uint16) (rapdu *apdu.RAPDU, err error) {
	return CommandNotAllowed(), nil
}

// GenerateAsymmetricKeyPair implements
// p72, 7.2.14 GENERATE ASYMMETRIC KEY PAIR, OpenPGP application Version 3.4.
//
// Generation of key pair is not implemented as card personalization is managed
// outside OpenPGP specifications and the PW3 PIN (required for this mode) is
// not  supported.
//
// Therefore this command can only be used to read public key templates.
func (card *Interface) GenerateAsymmetricKeyPair(params uint16, crt []byte) (rapdu *apdu.RAPDU, err error) {
	var subkey *openpgp.Subkey

	rapdu = CommandNotAllowed()

	switch params {
	case 0x8000:
		// Generation of key pair.
		//
		// Not implemented as card personalization is managed outside
		// OpenPGP specifications and therefore PW3 PIN is not
		// supported.
		return
	case 0x8100:
		// Reading of actual public key template.
	default:
		log.Printf("unsupported GENERATE parameters %x", params)
		return
	}

	switch {
	case bytes.Equal(crt, []byte{0xb6, 0x00}), bytes.Equal(crt, []byte{0xb6, 0x03, 0x84, 0x01, 0x01}):
		// Digital signature
		subkey = card.Sig
	case bytes.Equal(crt, []byte{0xb8, 0x00}), bytes.Equal(crt, []byte{0xb8, 0x03, 0x84, 0x01, 0x02}):
		// Confidentiality
		subkey = card.Dec
	case bytes.Equal(crt, []byte{0xa4, 0x00}), bytes.Equal(crt, []byte{0xa4, 0x03, 0x84, 0x01, 0x03}):
		// Authentication
		subkey = card.Aut
	default:
		log.Printf("unsupported GENERATE CRT %x", crt)
		return
	}

	if subkey == nil {
		return ReferencedDataNotFound(), nil
	}

	data := new(bytes.Buffer)

	switch pubKey := subkey.PublicKey.PublicKey.(type) {
	case *rsa.PublicKey:
		mod := pubKey.N.Bytes()
		exp := make([]byte, 4)
		binary.BigEndian.PutUint32(exp, uint32(pubKey.E))

		data.Write(tlv(DO_RSA_MOD, mod))
		data.Write(tlv(DO_RSA_EXP, exp))
	case *ecdsa.PublicKey:
		pp := elliptic.Marshal(pubKey, pubKey.X, pubKey.Y)
		data.Write(tlv(DO_EXT_PUB_KEY, pp))
	case *ecdh.PublicKey:
		pp := elliptic.Marshal(pubKey, pubKey.X, pubKey.Y)
		data.Write(tlv(DO_EXT_PUB_KEY, pp))
	default:
		err = fmt.Errorf("unexpected public key type in GENERATE %T", pubKey)
		return
	}

	return CommandCompleted(tlv(DO_PUB_KEY, data.Bytes())), nil
}
