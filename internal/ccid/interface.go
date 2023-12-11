// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package ccid

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/usbarmory/GoKey/internal/icc"
)

// p26, Table 6.1-1, CCID Rev1.1
const (
	ICC_POWER_ON     = 0x62
	ICC_POWER_OFF    = 0x63
	GET_SLOT_STATUS  = 0x65
	XFR_BLOCK        = 0x6f
	GET_PARAMETERS   = 0x6c
	RESET_PARAMETERS = 0x6d
	SET_PARAMETERS   = 0x61
)

// p48, Table 6.2-1, CCID Rev1.1
const (
	DATA_BLOCK  = 0x80
	SLOT_STATUS = 0x81
	PARAMETERS  = 0x82
)

// Interface implements a CCID compliant USB smartcard reader.
type Interface struct {
	ICC *icc.Interface
}

// CCIDCommand is the interface of individual CCID command handlers.
type CCIDCommand interface {
	Handle(buf []byte, card *icc.Interface) (res []byte, err error)
}

// Rx handles incoming CCID commands and invokes the relevant command handler.
func (ccid *Interface) Rx(buf []byte) (res []byte, err error) {
	var cmd CCIDCommand

	if len(buf) == 0 {
		return nil, errors.New("invalid CCID command, too short")
	}

	if buf[0] != GET_SLOT_STATUS {
		ccid.ICC.Wake()
	}

	switch buf[0] {
	case ICC_POWER_ON:
		cmd = &IccPowerOn{}
	case ICC_POWER_OFF:
		cmd = &IccPowerOff{}
	case GET_SLOT_STATUS:
		cmd = &GetSlotStatus{}
	case XFR_BLOCK:
		cmd = &XfrBlock{}
	case GET_PARAMETERS, RESET_PARAMETERS, SET_PARAMETERS:
		cmd = &GetParameters{}
	default:
		return nil, fmt.Errorf("invalid CCID command, unsupported: %x", buf)
	}

	if err = binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, cmd); err != nil {
		return
	}

	return cmd.Handle(buf, ccid.ICC)
}
