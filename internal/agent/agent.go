// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package agent

import (
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/usbarmory/GoKey/internal/led"
	"github.com/usbarmory/GoKey/internal/snvs"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	// user presence timeout in seconds
	timeout = 10
)

type Agent struct {
	// Presence is a channel used to signal user presence, when undefined
	// user presence is implicitly acknowledged.
	Presence chan bool

	agent agent.Agent
}

func (a *Agent) List() ([]*agent.Key, error) {
	return a.agent.List()
}

func (a *Agent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	if !a.UserPresence() {
		return nil, errors.New("user not present")
	}

	return a.agent.Sign(key, data)
}

func (a *Agent) Add(key agent.AddedKey) error {
	return a.agent.Add(key)
}

func (a *Agent) Remove(key ssh.PublicKey) error {
	return a.agent.Remove(key)
}

func (a *Agent) RemoveAll() error {
	return a.agent.RemoveAll()
}

func (a *Agent) Lock(passphrase []byte) error {
	return a.agent.Lock(passphrase)
}

func (a *Agent) Unlock(passphrase []byte) error {
	return a.agent.Unlock(passphrase)
}

func (a *Agent) Signers() ([]ssh.Signer, error) {
	return a.agent.Signers()
}

// UserPresence verifies the user presence.
func (a *Agent) UserPresence() (present bool) {
	if a.Presence == nil {
		return true
	}

	var done = make(chan bool)
	go led.Blink("white", done)

	log.Printf("agent user presence request, type `s` within %ds to confirm", timeout)

	select {
	case <-a.Presence:
		present = true
	case <-time.After(timeout * time.Second):
		log.Printf("agent user presence request timed out")
	}

	done <- true

	if present {
		log.Printf("agent user presence confirmed")
	}

	return
}

func (a *Agent) Init() (err error) {
	if !snvs.Available() {
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

	a.agent = sshAgent

	return
}

func (a *Agent) Identities() (res string, err error) {
	if a.agent == nil {
		if err = a.Init(); err != nil {
			return
		}
	}

	keys, err := a.agent.List()

	if err != nil {
		return
	}

	for _, key := range keys {
		res += fmt.Sprintf("%s", key)
	}

	return
}

func (a *Agent) Serve(rw io.ReadWriter) (err error) {
	if a.agent == nil {
		if err = a.Init(); err != nil {
			return
		}
	}

	return agent.ServeAgent(a, rw)
}
