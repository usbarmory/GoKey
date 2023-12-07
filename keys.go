// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

//go:generate go run bundle_keys.go

// OpenPGP
var (
	SNVS          bool
	initAtBoot    bool
	sshPublicKey  []byte
	sshPrivateKey []byte
	pgpSecretKey  []byte
	URL           string
	NAME          string
	LANGUAGE      string
	SEX           string
)

// U2F
var (
	u2fPublicKey  []byte
	u2fPrivateKey []byte
)
