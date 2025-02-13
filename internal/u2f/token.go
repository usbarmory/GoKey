// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package u2f

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"regexp"

	"github.com/usbarmory/GoKey/internal/snvs"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
	"github.com/usbarmory/tamago/soc/nxp/usb"

	"github.com/gsora/fidati"
	"github.com/gsora/fidati/attestation"
	"github.com/gsora/fidati/keyring"
	"github.com/gsora/fidati/u2fhid"
	"github.com/gsora/fidati/u2ftoken"
)

// Token represents an U2F authenticator instance.
type Token struct {
	// enable device unique hardware encryption for bundled private keys
	SNVS bool
	// Attestation certificate
	PublicKey []byte
	// Attestation private key
	PrivateKey []byte
	// Presence is a channel used to signal user presence, when undefined
	// user presence is implicitly acknowledged.
	Presence chan bool

	// Keyring instance
	keyring *keyring.Keyring
	// Monotonic counter instance
	counter *Counter

	// internal state flags
	initialized bool
}

// Configure initializes the U2F token attestation keys and USB configuration.
func Configure(device *usb.Device, token *Token) (err error) {
	token.keyring = &keyring.Keyring{}

	if token.SNVS {
		token.PrivateKey, err = snvs.Decrypt(token.PrivateKey, []byte(DiversifierU2F))

		if err != nil {
			return fmt.Errorf("U2F key decryption failed, %v", err)
		}
	}

	t, err := u2ftoken.New(token.keyring, token.PublicKey, token.PrivateKey)

	if err != nil {
		return
	}

	hid, err := u2fhid.NewHandler(t)

	if err != nil {
		return
	}

	if err = fidati.ConfigureUSB(device.Configurations[0], device, hid); err != nil {
		return
	}

	// resolve conflict with Ethernet over USB
	numInterfaces := len(device.Configurations[0].Interfaces)
	device.Configurations[0].Interfaces[numInterfaces-1].Endpoints[usb.OUT].EndpointAddress = 0x04
	device.Configurations[0].Interfaces[numInterfaces-1].Endpoints[usb.IN].EndpointAddress = 0x84

	return
}

// Init initializes an U2F authenticator instance.
func (token *Token) Init() (err error) {
	if token.keyring == nil {
		return errors.New("U2F token initialization failed, missing configuration")
	}

	counter := &Counter{}

	if err = counter.Init(token.Presence); err != nil {
		return
	}

	var mk []byte

	if token.SNVS {
		if mk, err = imx6ul.DCP.DeriveKey([]byte(DiversifierU2F), make([]byte, 16), -1); err != nil {
			return
		}
	} else {
		// On non-secure booted units we derive the master key from the
		// counter hardware serial number and the SoC unique ID.
		//
		// This provides a non-predictable master key which must
		// however be assumed compromised if a device is stolen/lost.
		uid := imx6ul.UniqueID()

		if mk, err = pbkdf2.Key(sha256.New, string(counter.Serial()), uid[:], 4096, 16); err != nil {
			return
		}
	}

	token.keyring.MasterKey = mk
	token.keyring.Counter = counter
	token.counter = counter
	token.initialized = true

	log.Printf("U2F token initialized")
	log.Print(token.Status())

	return
}

// Initialized returns the U2F token initialization state.
func (token *Token) Initialized() bool {
	return token.initialized
}

// Status returns attestation certificate fingerprint and counter status in
// textual format.
func (token *Token) Status() string {
	var status bytes.Buffer
	var s string

	fmt.Fprintf(&status, "------------------------------------------------------------ U2F token ----\n")
	fmt.Fprintf(&status, "Initialized ............: %v\n", token.initialized)
	fmt.Fprintf(&status, "Secure storage .........: %v\n", token.SNVS)
	fmt.Fprintf(&status, "User presence test .....: %v\n", token.Presence != nil)

	if token.initialized {
		val, err := token.counter.Read()

		if err != nil {
			s = err.Error()
		} else {
			s = fmt.Sprintf("%d", val)
		}
	} else {
		s = "N/A"
	}

	fmt.Fprintf(&status, "Counter ................: %v\n", s)
	fmt.Fprintf(&status, "Attestation certificate.: ")

	r := regexp.MustCompile(`([[:xdigit:]]{4})`)

	if k := token.PublicKey; len(k) != 0 {
		_, cert, err := attestation.ParseCertificate(k)

		if err != nil {
			s = err.Error()
		}

		fp := fmt.Sprintf("%X\n", sha1.Sum(cert.Raw))
		status.WriteString(r.ReplaceAllString(fp, "$1 "))
	} else {
		status.WriteString("missing\n")
	}

	return status.String()
}
