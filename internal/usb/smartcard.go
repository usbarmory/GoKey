// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package usb

import (
	"github.com/usbarmory/GoKey/internal/ccid"

	"github.com/usbarmory/tamago/soc/nxp/usb"
)

var queue = make(chan []byte)
var CCID *ccid.Interface

// CCIDTx implements the endpoint 1 IN function, used to transmit APDU
// responses from device to host.
func CCIDTx(_ []byte, lastErr error) (in []byte, err error) {
	select {
	case res := <-queue:
		in = res
	default:
	}

	return
}

// CCIDRx implements the endpoint 1 OUT function, used to receive APDU
// requests from host to device.
func CCIDRx(out []byte, lastErr error) (_ []byte, err error) {
	in, err := CCID.Rx(out)

	if err != nil {
		return
	}

	queue <- in

	return
}

// ConfigureCCID configures a Chip/SmartCard interface USB device.
func ConfigureCCID(device *usb.Device, ccidInterface *ccid.Interface) {
	CCID = ccidInterface

	// Chip/SmartCard interface
	iface := &usb.InterfaceDescriptor{}
	iface.SetDefaults()
	iface.NumEndpoints = 2
	iface.InterfaceClass = usb.SMARTCARD_DEVICE_CLASS

	iInterface, _ := device.AddString(`Smart Card Control`)
	iface.Interface = iInterface

	ccid := &usb.CCIDDescriptor{}
	ccid.SetDefaults()
	// all voltages
	ccid.VoltageSupport = 0x7
	// T=0, T=1
	ccid.Protocols = 0x3
	// 4 MHz
	ccid.DefaultClock = 4000
	// 5 MHz
	ccid.MaximumClock = 5000
	ccid.DataRate = 9600
	// maximum@5MHz according to ISO7816-3
	ccid.MaxDataRate = 625000
	ccid.MaxIFSD = 0xfe
	// Auto configuration based on ATR
	// Auto activation on insert
	// Auto voltage selection
	// Auto clock change
	// Auto baud rate change
	// Auto parameter negotiation made by CCID
	// Short and extended APDU level exchange
	ccid.Features = 0x02 | 0x04 | 0x08 | 0x10 | 0x20 | 0x40 | 0x40000
	ccid.MaxCCIDMessageLength = usb.DTD_PAGES * usb.DTD_PAGE_SIZE
	ccid.MaxIFSD = ccid.MaxCCIDMessageLength
	// echo
	ccid.ClassGetResponse = 0xff
	ccid.ClassEnvelope = 0xff
	ccid.MaxCCIDBusySlots = 1

	iface.ClassDescriptors = append(iface.ClassDescriptors, ccid.Bytes())

	ep3IN := &usb.EndpointDescriptor{}
	ep3IN.SetDefaults()
	ep3IN.EndpointAddress = 0x83
	ep3IN.Attributes = 2
	ep3IN.MaxPacketSize = maxPacketSize
	ep3IN.Function = CCIDTx

	iface.Endpoints = append(iface.Endpoints, ep3IN)

	ep3OUT := &usb.EndpointDescriptor{}
	ep3OUT.SetDefaults()
	ep3OUT.EndpointAddress = 0x03
	ep3OUT.Attributes = 2
	ep3OUT.MaxPacketSize = maxPacketSize
	ep3OUT.Function = CCIDRx

	iface.Endpoints = append(iface.Endpoints, ep3OUT)

	device.Configurations[configurationIndex].AddInterface(iface)
}
