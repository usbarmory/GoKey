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

	imxusb "github.com/f-secure-foundry/tamago/soc/imx6/usb"
	"github.com/f-secure-foundry/tamago/soc/imx6/usb/ethernet"
)

const (
	hostMAC   = "1a:55:89:a2:69:42"
	deviceMAC = "1a:55:89:a2:69:41"
	IP        = "10.0.0.10"
)

func startNetworking(device *imxusb.Device, card *icc.Interface) {
	// start basic networking
	stack, link := gokey.StartNetworking(deviceMAC, IP)

	if len(sshPublicKey) != 0 {
		started := make(chan bool)

		// start SSH server for management console
		err := gokey.StartSSHServer(stack, IP, sshPublicKey, sshPrivateKey, card, started)

		if err != nil {
			log.Printf("SSH server initialization error: %v", err)
		}

		// wait for ssh server to start before responding to USB requests
		<-started
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
