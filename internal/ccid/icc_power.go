// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package ccid

import (
	"github.com/usbarmory/GoKey/internal/icc"
)

// IccPowerOn implements p26, 6.1.1 PC_to_RDR_IccPowerOn, CCID Rev1.1.
type IccPowerOn struct {
	MessageType uint8
	Length      uint32
	Slot        uint8
	Seq         uint8
	PowerSelect uint8
	RFU         [2]byte
}

// IccPowerOff implements p28, 6.1.2 PC_to_RDR_IccPowerOff, CCID Rev1.1.
type IccPowerOff struct {
	MessageType uint8
	Length      uint32
	Slot        uint8
	Seq         uint8
	RFU         [3]byte
}

// Handle ICC power on requests by returning the ATR.
func (cmd *IccPowerOn) Handle(_ []byte, card *icc.Interface) (buf []byte, err error) {
	res := &DataBlock{
		MessageType: DATA_BLOCK,
		Slot:        cmd.Slot,
		Seq:         cmd.Seq,
	}

	atr := card.ATR()
	res.Length = uint32(len(atr))

	if buf, err = Serialize(res); err != nil {
		return
	}

	buf = append(buf, atr...)

	return
}

// Handle ICC power off requests (NOP, card always active).
func (cmd *IccPowerOff) Handle(_ []byte, _ *icc.Interface) ([]byte, error) {
	res := &SlotStatus{
		MessageType: SLOT_STATUS,
		Slot:        cmd.Slot,
		Seq:         cmd.Seq,
	}

	return Serialize(res)
}
