// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package usb

import (
	"log"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"
	"github.com/usbarmory/tamago/soc/nxp/usb"
)

func StartInterruptHandler(port *usb.USB) {
	if port == nil {
		return
	}

	imx6ul.GIC.Init()
	imx6ul.GIC.EnableInterrupt(port.IRQ)

	port.EnableInterrupt(usb.IRQ_URI) // reset
	port.EnableInterrupt(usb.IRQ_PCI) // port change detect
	port.EnableInterrupt(usb.IRQ_UI)  // transfer completion

	isr := func() {
		irq := imx6ul.GIC.GetInterrupt()

		switch irq {
		case port.IRQ:
			port.ServiceInterrupts()
		default:
			log.Printf("internal error, unexpected IRQ %d", irq)
		}
	}

	imx6ul.ARM.ServiceInterrupts(isr)
}
