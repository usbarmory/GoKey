// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

//go:generate go run bundle_keys.go

// SNVS
var (
	SNVS       bool
	initAtBoot bool
)

// SSH
var (
	sshPublicKey  []byte
	sshPrivateKey []byte
)

// OpenPGP
var (
	pgpSecretKey []byte
	URL          string
	NAME         string
	LANGUAGE     string
	SEX          string
)

// U2F
var (
	u2fPublicKey  []byte
	u2fPrivateKey []byte
)
