// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package usb

import (
	"github.com/f-secure-foundry/GoKey/internal/ccid"

	"github.com/f-secure-foundry/tamago/imx6"
	"github.com/f-secure-foundry/tamago/imx6/usb"
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
	imx6.SetARMFreq(900);
	defer imx6.SetARMFreq(198);

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
	iface.InterfaceClass = 0xb

	iInterface, _ := device.AddString(`Smart Card Control`)
	iface.Interface = iInterface

	// Set IAD to be inserted before first interface, to support multiple
	// functions in this same configuration.
	iface.IAD = &usb.InterfaceAssociationDescriptor{}
	iface.IAD.SetDefaults()
	iface.IAD.InterfaceCount = 1
	iface.IAD.FunctionClass = iface.InterfaceClass

	iFunction, _ := device.AddString(`CCID`)
	iface.IAD.Function = iFunction

	ccid := &usb.CCIDDescriptor{}
	ccid.SetDefaults()

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
