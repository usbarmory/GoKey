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
	"crypto/sha256"
	"errors"
	"fmt"
	"log"

	"golang.org/x/crypto/pbkdf2"

	"github.com/f-secure-foundry/GoKey/internal/snvs"

	"github.com/f-secure-foundry/tamago/soc/imx6"
	"github.com/f-secure-foundry/tamago/soc/imx6/usb"

	"github.com/gsora/fidati"
	"github.com/gsora/fidati/keyring"
	"github.com/gsora/fidati/u2fhid"
	"github.com/gsora/fidati/u2ftoken"
)

// Token represents an U2F authenticator instance.
type Token struct {
	// Attestation certificate
	PublicKey []byte
	// Attestation private key
	PrivateKey []byte
	// Keyring instance
	Keyring *keyring.Keyring
	// Monotonic counter instance
	Counter *Counter
	// Presence is a channel used to signal user presence, when undefined
	// user presence is implicitly acknowledged.
	Presence chan bool
}

// Configure initializes the U2F token attestation keys and USB configuration.
func Configure(device *usb.Device, token *Token, SNVS bool) (err error) {
	token.Keyring = &keyring.Keyring{}

	if SNVS {
		token.PrivateKey, err = snvs.Decrypt(token.PrivateKey, []byte(DiversifierU2F))

		if err != nil {
			return fmt.Errorf("U2F key decryption failed, %v", err)
		}
	}

	t, err := u2ftoken.New(token.Keyring, token.PublicKey, token.PrivateKey)

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
	if token.Keyring == nil {
		return errors.New("U2F token initialization failed, missing configuration")
	}

	token.Counter = &Counter{}
	cnt, err := token.Counter.Init(token.Presence)

	if err != nil {
		return
	}

	var mk []byte

	if imx6.DCP.SNVS() {
		mk, err = imx6.DCP.DeriveKey([]byte(DiversifierU2F), make([]byte, 16), -1)

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
		sn, err := token.Counter.Info()

		if err != nil {
			return err
		}

		mk = pbkdf2.Key([]byte(sn), uid[:], 4096, 16, sha256.New)
	}

	token.Keyring.MasterKey = mk
	token.Keyring.Counter = token.Counter

	log.Printf("U2F token initialized, managed:%v counter:%d", (token.Presence != nil), cnt)

	return
}
