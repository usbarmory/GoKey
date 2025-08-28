// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package age

import (
	"crypto/aes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"

	"filippo.io/age"
	"filippo.io/age/plugin"

	"github.com/usbarmory/GoKey/internal/snvs"
)

// Diversifier for hardware key derivation (age private key wrapping).
const DiversifierAGE = "GoKeySNVSAGE    "

type Plugin struct {
	// enable device unique hardware encryption for bundled private keys
	SNVS bool

	// age plugin instance
	plugin *plugin.Plugin

	// internal state flags
	initialized bool
}

func (p *Plugin) generate() (res string, err error) {
	identity, err := age.GenerateX25519Identity()

	if err != nil {
		return
	}

	iv := make([]byte, aes.BlockSize)

	if _, err = rand.Read(iv); err != nil {
		return
	}

	output, err := snvs.Encrypt([]byte(identity.String()), []byte(DiversifierAGE), iv)

	if err != nil {
		return
	}

	res = fmt.Sprintf("# public key: %s\n%s",
		identity.Recipient().String(),
		plugin.EncodeIdentity("GOKEY", output),
	)

	return
}

func (p *Plugin) identity(rw io.ReadWriter) (err error) {
	p.plugin.SetIO(rw, rw, log.Default().Writer())

	if c := p.plugin.IdentityV1(); c != 0 {
		return fmt.Errorf("exit code: %v", c)
	}

	return
}

func (p *Plugin) Init() (err error) {
	ap, err := plugin.New("gokey")

	if err != nil {
		return
	}

	ap.HandleRecipient(func(data []byte) (age.Recipient, error) {
		return nil, errors.New("unsupported")
	})

	ap.HandleIdentity(func(data []byte) (age.Identity, error) {
		log.Printf("age/plugin identity-v1 data:%x", data)

		s, err := snvs.Decrypt(data, []byte(DiversifierAGE))

		if err != nil {
			return nil, err
		}

		return age.ParseX25519Identity(string(s))
	})

	p.plugin = ap
	p.initialized = true

	return
}

// Initialized returns the age plugin initialization state.
func (p *Plugin) Initialized() bool {
	return p.initialized
}

func (p *Plugin) Handle(rw io.ReadWriter, sm string) (res string) {
	var err error

	if !p.SNVS {
		return "SNVS not available"
	}

	switch sm {
	case "gen":
		res, err = p.generate()
	case "identity-v1":
		err = p.identity(rw)
	default:
		return "unsupported"
	}

	if err != nil {
		return err.Error()
	}

	return
}
