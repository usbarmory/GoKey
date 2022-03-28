// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package ccid

import (
	"github.com/usbarmory/GoKey/internal/icc"
)

const (
	// p55, Table 6.2-3 Slot Status register, CCID Rev1.1
	ICC_PRESENT_AND_ACTIVE = 0
	FAILED                 = 1 << 6
)

// GetSlotStatus implements p29, 6.1.3 PC_to_RDR_GetSlotStatus, CCID Rev1.1.
type GetSlotStatus struct {
	MessageType uint8
	Length      uint32
	Slot        uint8
	Seq         uint8
	RFU         [3]byte
}

// SlotStatus implements p50, 6.2.2 RDR_to_PC_SlotStatus, CCID Rev1.1.
type SlotStatus struct {
	MessageType uint8
	Length      uint32
	Slot        uint8
	Seq         uint8
	Status      uint8
	Error       uint8
	ClockStatus uint8
}

// Handle slot status requests (NOP, card always active).
func (cmd *GetSlotStatus) Handle(_ []byte, _ *icc.Interface) ([]byte, error) {
	res := &SlotStatus{
		MessageType: SLOT_STATUS,
		Slot:        cmd.Slot,
		Seq:         cmd.Seq,
		Status:      ICC_PRESENT_AND_ACTIVE,
	}

	return Serialize(res)
}
