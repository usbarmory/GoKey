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
	"log"
	"net"

	"github.com/f-secure-foundry/GoKey/internal"
	"github.com/f-secure-foundry/GoKey/internal/icc"
	"github.com/f-secure-foundry/GoKey/internal/u2f"

	imxusb "github.com/f-secure-foundry/tamago/soc/imx6/usb"
	"github.com/f-secure-foundry/tamago/soc/imx6/usb/ethernet"
)

const (
	hostMAC   = "1a:55:89:a2:69:42"
	deviceMAC = "1a:55:89:a2:69:41"
	IP        = "10.0.0.10"
)

func startNetworking(device *imxusb.Device, card *icc.Interface, token *u2f.Token) {
	// start basic networking
	stack, link := gokey.StartNetworking(deviceMAC, IP)

	if len(sshPublicKey) != 0 {
		console := &gokey.Console{
			Stack:         stack,
			Interface:     1,
			Address:       IP,
			Port:          22,
			AuthorizedKey: sshPublicKey,
			PrivateKey:    sshPrivateKey,
			Card:          card,
			Token:         token,
			Started:       make(chan bool),
		}

		// start SSH server for management console
		err := console.Start()

		if err != nil {
			log.Printf("SSH server initialization error: %v", err)
		}

		// wait for ssh server to start before responding to USB requests
		<-console.Started
	}

	hostAddress, err := net.ParseMAC(hostMAC)

	if err != nil {
		log.Fatal(err)
	}

	deviceAddress, err := net.ParseMAC(deviceMAC)

	if err != nil {
		log.Fatal(err)
	}

	// Configure Ethernet over USB endpoints
	// (ECM protocol, only supported on Linux hosts).
	eth := ethernet.NIC{
		Host:   hostAddress,
		Device: deviceAddress,
		Link:   link,
	}

	err = eth.Init(device, 0)

	if err != nil {
		log.Fatal(err)
	}
}
