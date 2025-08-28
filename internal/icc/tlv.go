// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"bytes"
	"encoding/binary"
)

func tlv(t int, v []byte) (buf []byte) {
	l := len(v)

	if t > 0xff {
		// 2nd byte is tag value
		buf = make([]byte, 2)
		binary.BigEndian.PutUint16(buf, uint16(t))
	} else {
		buf = append(buf, byte(t))
	}

	// p39, 4.4.4 Length Field of DOs, OpenPGP application Version 3.4
	if l <= 0x7f {
		buf = append(buf, uint8(l))
	} else if len(v) <= 0xff {
		buf = append(buf, 0x81)
		buf = append(buf, uint8(l))
	} else {
		buf = append(buf, 0x82)
		buf = append(buf, byte((l>>8)&0xff))
		buf = append(buf, byte(l&0xff))
	}

	buf = append(buf, v...)

	return
}

func v(buf []byte, t int) (v []byte) {
	var i int
	var l int

	if len(buf) < 1 {
		return
	}

	if t > 0xff {
		// 2nd byte is tag value
		if len(buf) < 2 {
			return
		}

		tag := make([]byte, 2)
		binary.BigEndian.PutUint16(tag, uint16(t))

		if !bytes.Equal(tag, buf[0:2]) {
			return
		}

		i += 2
	} else {
		if byte(t) != buf[0] {
			return
		}

		i += 1
	}

	if buf[i] <= 0x7f {
		l = int(buf[i])
		i += 1
	} else if buf[i] == 0x81 {
		if len(buf) < i+1 {
			return
		}

		l = int(buf[i+1])
		i += 2
	} else if buf[i] == 0x82 {
		if len(buf) < i+2 {
			return
		}

		l = int(binary.BigEndian.Uint16(buf[i+1 : i+2]))
		i += 2
	}

	if len(buf) < i+l {
		return
	}

	return buf[i : i+l]
}
