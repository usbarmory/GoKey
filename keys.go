// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

//go:generate go run bundle_keys.go

// OpenPGP
var SNVS bool
var initAtBoot bool
var sshPublicKey []byte
var sshPrivateKey []byte
var pgpSecretKey []byte
var URL string
var NAME string
var LANGUAGE string
var SEX string

// U2F
var u2fPublicKey []byte
var u2fPrivateKey []byte
