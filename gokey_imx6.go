// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/usbarmory/GoKey/internal/ccid"
	"github.com/usbarmory/GoKey/internal/icc"
	"github.com/usbarmory/GoKey/internal/u2f"
	"github.com/usbarmory/GoKey/internal/usb"

	"github.com/usbarmory/tamago/soc/imx6"
	imxusb "github.com/usbarmory/tamago/soc/imx6/usb"

	usbarmory "github.com/usbarmory/tamago/board/usbarmory/mk2"

	"github.com/usbarmory/imx-usbnet"
)

const (
	deviceIP  = "10.0.0.10"
	deviceMAC = "1a:55:89:a2:69:41"
	hostMAC   = "1a:55:89:a2:69:42"
)

// initialized at compile time (see Makefile)
var Build string
var Revision string

func init() {
	if err := imx6.SetARMFreq(imx6.FreqMax); err != nil {
		panic(fmt.Sprintf("WARNING: error setting ARM frequency: %v\n", err))
	}
}

func initCard(device *imxusb.Device, card *icc.Interface) {
	// Initialize an OpenPGP card with the bundled key information (defined
	// in `keys.go` and generated at compilation time).
	card.SNVS = SNVS
	card.ArmoredKey = pgpSecretKey
	card.Name = NAME
	card.Language = LANGUAGE
	card.Sex = SEX
	card.URL = URL
	card.Debug = false

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

	// set card serial number to 2nd half of NXP Unique ID
	uid := imx6.UniqueID()
	copy(card.Serial[0:4], uid[4:8])

	// configure Smart Card over USB endpoints (CCID protocol)
	usb.ConfigureCCID(device, reader)
}

func initToken(device *imxusb.Device, token *u2f.Token) {
	token.SNVS = SNVS
	token.PublicKey = u2fPublicKey
	token.PrivateKey = u2fPrivateKey

	err := u2f.Configure(device, token)

	if err != nil {
		log.Printf("U2F configuration error: %v", err)
	}

	if initAtBoot {
		err = token.Init()

		if err != nil {
			log.Printf("U2F initialization error: %v", err)
		}
	}
}

func main() {
	device := &imxusb.Device{}
	card := &icc.Interface{}
	token := &u2f.Token{}

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	usb.ConfigureDevice(device)

	if SNVS && !imx6.SNVS() {
		log.Fatalf("SNVS not available")
	}

	if len(pgpSecretKey) != 0 {
		initCard(device, card)
	}

	if len(u2fPublicKey) != 0 && len(u2fPrivateKey) != 0 {
		initToken(device, token)
	}

	if len(sshPublicKey) != 0 {
		configureNetworking(device, card, token)
	}

	// The plug is checked, rather than the receptacle, as a workaround for:
	// https://github.com/usbarmory/usbarmory/wiki/Errata-(Mk-II)#errata-type-c-plugreceptacle-reset-plug-resolved-receptacle-workaround
	mode, _ := usbarmory.PlugMode()
	port := imxusb.USB1

	if mode == usbarmory.STATE_NOT_ATTACHED {
		port = imxusb.USB2
	}

	port.Init()
	port.DeviceMode()
	port.Reset()

	if err := imx6.SetARMFreq(imx6.FreqLow); err != nil {
		log.Fatalf("WARNING: error setting ARM frequency: %v\n", err)
	}

	// never returns
	port.Start(device)
}

func configureNetworking(device *imxusb.Device, card *icc.Interface, token *u2f.Token) {
	gonet, err := usbnet.Add(device, deviceIP, deviceMAC, hostMAC, 1)

	if err != nil {
		log.Fatalf("could not initialize USB networking, %v", err)
	}

	gonet.EnableICMP()

	listener, err := gonet.ListenerTCP4(22)

	if err != nil {
		log.Fatalf("could not initialize SSH listener, %v", err)
	}

	banner := fmt.Sprintf("GoKey • %s/%s (%s) • %s %s",
		runtime.GOOS, runtime.GOARCH, runtime.Version(), Revision, Build)

	console := &usb.Console{
		AuthorizedKey: sshPublicKey,
		PrivateKey:    sshPrivateKey,
		Card:          card,
		Token:         token,
		Started:       make(chan bool),
		Listener:      listener,
		Banner:        banner,
	}

	// start SSH server for management console
	err = console.Start()

	if err != nil {
		log.Printf("SSH server initialization error: %v", err)
	}

	// wait for ssh server to start before responding to USB requests
	<-console.Started
}
