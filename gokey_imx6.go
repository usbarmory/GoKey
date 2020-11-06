// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package main

import (
	"fmt"
	"log"

	"github.com/f-secure-foundry/GoKey/internal"
	"github.com/f-secure-foundry/GoKey/internal/ccid"
	"github.com/f-secure-foundry/GoKey/internal/icc"
	"github.com/f-secure-foundry/GoKey/internal/u2f"
	"github.com/f-secure-foundry/GoKey/internal/usb"

	"github.com/f-secure-foundry/tamago/soc/imx6"
	imxusb "github.com/f-secure-foundry/tamago/soc/imx6/usb"

	_ "github.com/f-secure-foundry/tamago/board/f-secure/usbarmory/mark-two"
)

func init() {
	if !imx6.Native {
		return
	}

	if err := imx6.SetARMFreq(900); err != nil {
		panic(fmt.Sprintf("WARNING: error setting ARM frequency: %v\n", err))
	}
}

func main() {
	log.Println(gokey.Banner)

	device := &imxusb.Device{}
	usb.ConfigureDevice(device)

	// Initialize an OpenPGP card with the bundled key information (defined
	// in `keys.go` and generated at compilation time).
	card := &icc.Interface{
		SNVS:       SNVS,
		ArmoredKey: pgpSecretKey,
		Name:       NAME,
		Language:   LANGUAGE,
		Sex:        SEX,
		URL:        URL,
		Debug:      false,
	}

	if initAtBoot {
		err := card.Init()

		if err != nil {
			log.Printf("OpenPGP ICC initialization error: %v", err)
		}
	}

	// initialize CCID interface
	reader := &ccid.Interface{
		ICC: card,
	}

	if imx6.Native {
		// set card serial number to 2nd half of NXP Unique ID
		uid := imx6.UniqueID()
		copy(card.Serial[0:4], uid[4:8])

		// configure Smart Card over USB endpoints (CCID protocol)
		usb.ConfigureCCID(device, reader)
	} else {
		return
	}

	if len(sshPublicKey) != 0 {
		startNetworking(device, card)
	}

	if len(u2fPublicKey) != 0 && len(u2fPrivateKey) != 0 {
		err := u2f.Configure(device, u2fPublicKey, u2fPrivateKey)

		if err != nil {
			log.Printf("U2F initialization error: %v", err)
		}
	}

	imxusb.USB1.Init()
	imxusb.USB1.DeviceMode()
	imxusb.USB1.Reset()

	if err := imx6.SetARMFreq(198); err != nil {
		log.Fatalf("WARNING: error setting ARM frequency: %v\n", err)
	}

	// never returns
	imxusb.USB1.Start(device)
}
