// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package ccid

import (
	"bytes"
	"encoding/binary"
)

// Serialize a structure to a byte array.
func Serialize(data interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, data)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Data extracts the abData field contents.
func Data(buf []byte, length uint32) (data []byte) {
	if length == 0 {
		return
	}

	if off := len(buf) - int(length); off > 0 {
		data = buf[off:]
	}

	return
}
