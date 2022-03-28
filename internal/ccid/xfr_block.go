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
	BAD_LEVEL_PARAMETER = 8
)

// XfrBlock implements p30, 6.1.4 PC_to_RDR_XfrBlock, CCID Rev1.1.
type XfrBlock struct {
	MessageType    uint8
	Length         uint32
	Slot           uint8
	Seq            uint8
	BWI            uint8
	LevelParameter uint16
}

// Handle APDU transfer requests.
func (cmd *XfrBlock) Handle(buf []byte, card *icc.Interface) (resBuf []byte, err error) {
	res := &DataBlock{
		MessageType: DATA_BLOCK,
		Slot:        cmd.Slot,
		Seq:         cmd.Seq,
	}

	if cmd.LevelParameter != 0 {
		res.Status = FAILED
		res.Error = BAD_LEVEL_PARAMETER
		return Serialize(res)
	}

	resData, err := card.RawCommand(Data(buf, cmd.Length))

	if err != nil {
		return
	}

	res.Length = uint32(len(resData))

	resBuf, err = Serialize(res)
	resBuf = append(resBuf, resData...)

	return
}
