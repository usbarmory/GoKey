// https://github.com/usbarmory/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package ccid

const MAX_DATA_BLOCK = 65538

// DataBlock p49, 6.2.1 RDR_to_PC_DataBlock, CCID Rev1.1.
type DataBlock struct {
	MessageType    uint8
	Length         uint32
	Slot           uint8
	Seq            uint8
	Status         uint8
	Error          uint8
	ChainParameter uint8
}
