// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"errors"
	"fmt"
	"io"

	"github.com/usbarmory/GoKey/internal/snvs"

	"golang.org/x/crypto/ssh/agent"
)

func (card *Interface) initAgent() (err error) {
	if !card.SNVS {
		return errors.New("SNVS not available")
	}

	deviceKey, err := snvs.DeviceKey(snvs.DiversifierSSH)

	if err != nil {
		return
	}

	sshAgent := agent.NewKeyring()

	if err = sshAgent.Add(agent.AddedKey{
		PrivateKey: deviceKey,
	}); err != nil {
		return
	}

	card.agent = sshAgent

	return
}

func (card *Interface) Identities() (res string, err error) {
	if card.agent == nil {
		if err = card.initAgent(); err != nil {
			return
		}
	}

	keys, err := card.agent.List()

	if err != nil {
		return
	}

	for _, key := range keys {
		res += fmt.Sprintf("%s", key)
	}

	return
}

func (card *Interface) ServeAgent(rw io.ReadWriter) (err error) {
	if card.agent == nil {
		if err = card.initAgent(); err != nil {
			return
		}
	}

	return agent.ServeAgent(card.agent, rw)
}
