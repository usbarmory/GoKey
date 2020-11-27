// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package u2f

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"regexp"

	"golang.org/x/crypto/pbkdf2"

	"github.com/f-secure-foundry/GoKey/internal/snvs"

	"github.com/f-secure-foundry/tamago/soc/imx6"
	"github.com/f-secure-foundry/tamago/soc/imx6/dcp"
	"github.com/f-secure-foundry/tamago/soc/imx6/usb"

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

	err = fidati.ConfigureUSB(device.Configurations[0], device, hid)

	if err != nil {
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
	err = counter.Init(token.Presence)

	if err != nil {
		return
	}

	var mk []byte

	if token.SNVS || imx6.SNVS() {
		mk, err = dcp.DeriveKey([]byte(DiversifierU2F), make([]byte, 16), -1)

		if err != nil {
			return
		}
	} else {
		// On non-secure booted units we derive the master key from the
		// ATECC608A security element random S/N and the SoC unique ID.
		//
		// This provides a non-predictable master key which must
		// however be assumed compromised if a device is stolen/lost.
		uid := imx6.UniqueID()
		sn, err := counter.Info()

		if err != nil {
			return err
		}

		mk = pbkdf2.Key([]byte(sn), uid[:], 4096, 16, sha256.New)
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

	status.WriteString("------------------------------------------------------------ U2F token ----\n")
	status.WriteString(fmt.Sprintf("Initialized ............: %v\n", token.initialized))
	status.WriteString(fmt.Sprintf("Secure storage .........: %v\n", token.SNVS))
	status.WriteString(fmt.Sprintf("User presence test .....: %v\n", token.Presence != nil))

	val, err := token.counter.Read()

	if err != nil {
		s = err.Error()
	} else {
		s = fmt.Sprintf("%d", val)
	}

	status.WriteString(fmt.Sprintf("Counter ................: %v\n", s))
	status.WriteString("Attestation certificate.: ")

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
