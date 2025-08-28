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

// GetParameters implements p31, 6.1.5 PC_to_RDR_GetParameters, CCID Rev1.1.
type GetParameters struct {
	MessageType uint8
	Length      uint32
	Slot        uint8
	Seq         uint8
	RFU         [3]byte
}

// Parameters implements p51, 6.2.3 RDR_to_PC_Parameters, CCID Rev1.1.
type Parameters struct {
	MessageType uint8
	Length      uint32
	Slot        uint8
	Seq         uint8
	Status      uint8
	Error       uint8
	ProtocolNum uint8
}

// Handle get/reset/set parameters requests.
func (cmd *GetParameters) Handle(_ []byte, _ *icc.Interface) ([]byte, error) {
	res := &Parameters{
		MessageType: PARAMETERS,
		Slot:        cmd.Slot,
		Seq:         cmd.Seq,
		Status:      ICC_PRESENT_AND_ACTIVE,
		ProtocolNum: 0x01, // indicate use of T=1
	}

	return Serialize(res)
}
