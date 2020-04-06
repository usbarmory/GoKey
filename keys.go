// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package main

//go:generate go run bundle_keys.go

//lint:ignore U1000 defined through go:generate
var SNVS bool

//lint:ignore U1000 defined through go:generate
var initAtBoot bool

//lint:ignore U1000 defined through go:generate
var sshPublicKey []byte

//lint:ignore U1000 defined through go:generate
var sshPrivateKey []byte

//lint:ignore U1000 defined through go:generate
var pgpSecretKey []byte

//lint:ignore U1000 defined through go:generate
var URL string

//lint:ignore U1000 defined through go:generate
var NAME string

//lint:ignore U1000 defined through go:generate
var LANGUAGE string

//lint:ignore U1000 defined through go:generate
var SEX string
