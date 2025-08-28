// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package usb

import (
	"github.com/usbarmory/tamago/soc/nxp/usb"
)

const configurationIndex = 0
const maxPacketSize = 512

// ConfigureDevice configures a basic composite USB device.
func ConfigureDevice(device *usb.Device, serial string) {
	// Supported Language Code Zero: English
	device.SetLanguageCodes([]uint16{0x0409})

	// device descriptor
	device.Descriptor = &usb.DeviceDescriptor{}
	device.Descriptor.SetDefaults()

	// p5, Table 1-1. Device Descriptor Using Class Codes for IAD,
	// USB Interface Association Descriptor Device Class Code and Use Model.
	device.Descriptor.DeviceClass = 0xef
	device.Descriptor.DeviceSubClass = 0x02
	device.Descriptor.DeviceProtocol = 0x01

	// http://pid.codes/1209/2702/
	device.Descriptor.VendorId = 0x1209
	device.Descriptor.ProductId = 0x2702

	device.Descriptor.Device = 0x0001

	// historical value kept to avoid caching issues
	iManufacturer, _ := device.AddString(`WithSecure Foundry`)
	device.Descriptor.Manufacturer = iManufacturer

	iProduct, _ := device.AddString(`Composite Ethernet ECM / OpenPGP Smart Card Device`)
	device.Descriptor.Product = iProduct

	iSerial, _ := device.AddString(serial)
	device.Descriptor.SerialNumber = iSerial

	conf := &usb.ConfigurationDescriptor{}
	conf.SetDefaults()

	device.AddConfiguration(conf)

	// device qualifier
	device.Qualifier = &usb.DeviceQualifierDescriptor{}
	device.Qualifier.SetDefaults()
	device.Qualifier.NumConfigurations = uint8(len(device.Configurations))
}
