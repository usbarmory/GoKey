// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"time"

	"github.com/google/go-p11-kit/p11kit"
)

func (card *Interface) getObjects() (objs []p11kit.Object, err error) {
	subkey := card.Sig

	if subkey == nil || subkey.PrivateKey == nil {
		log.Printf("missing private key for PKCS#11 slot object")
		return nil, errors.New("card key not supported")
	}

	if subkey.PrivateKey.Encrypted {
		log.Printf("security condition not satisfied key for PKCS#11 slot object")
		return nil, errors.New("security condition not satisfied")
	}

	if PW1_CDS_MULTI == 0 {
		defer card.Verify(PW_LOCK, PW1_CDS, nil)
	}

	creationTime := subkey.PrivateKey.CreationTime

	certTemplate := x509.Certificate{
		NotBefore: creationTime,
		// no well-defined expiration date
		NotAfter:     time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
		SerialNumber: big.NewInt(int64(subkey.PrivateKey.KeyId)),
		Subject: pkix.Name{
			CommonName: card.Name,
		},
	}

	defer func() {
		if err != nil {
			log.Printf("error, %+v", err)
		}
	}()

	switch privKey := subkey.PrivateKey.PrivateKey.(type) {
	case *rsa.PrivateKey:
		der, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &privKey.PublicKey, privKey)

		if err != nil {
			return nil, err
		}

		cert, err := x509.ParseCertificate(der)

		if err != nil {
			return nil, err
		}

		obj, err := p11kit.NewX509CertificateObject(cert)

		if err != nil {
			return nil, err
		}

		objs = append(objs, obj)

		if obj, err = p11kit.NewPrivateKeyObject(privKey); err != nil {
			return nil, err
		}

		objs = append(objs, obj)

		if obj, err = p11kit.NewPublicKeyObject(privKey.Public()); err != nil {
			return nil, err
		}

		if err = obj.SetCertificate(cert); err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	case *ecdsa.PrivateKey:
		obj, err := p11kit.NewPrivateKeyObject(privKey)

		if err != nil {
			return nil, err
		}

		objs = append(objs, obj)

		if obj, err = p11kit.NewPublicKeyObject(privKey.Public()); err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	default:
		log.Printf("card key not satisfied for PKCS#11 slot object")
		return nil, errors.New("card key not supported")
	}

	return
}

func (card *Interface) initRPC() {
	slot := p11kit.Slot{
		ID:              0x01,
		Description:     "GoKey PKCS#11 RPC",
		Label:           "GoKey",
		// historical value kept to avoid caching issues
		Manufacturer:    "WithSecure Foundry",
		Model:           "USB armory Mk II",
		Serial:          fmt.Sprintf("%X", card.Serial),
		HardwareVersion: p11kit.Version{Major: 2, Minor: 0},
		FirmwareVersion: p11kit.Version{Major: 0, Minor: 1},
		GetObjects:      card.getObjects,
	}

	card.rpc = &p11kit.Handler{
		// historical value kept to avoid caching issues
		Manufacturer:   "WithSecure Foundry",
		Library:        "GoKey",
		LibraryVersion: p11kit.Version{Major: 0, Minor: 1},
		Slots:          []p11kit.Slot{slot},
	}
}

func (card *Interface) ServeRPC(rw io.ReadWriter) error {
	if card.rpc == nil {
		card.initRPC()
	}

	return card.rpc.Handle(rw)
}
