// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package icc

import (
	"github.com/hsanjuan/go-nfctype4/apdu"
)

func CommandNotAllowed() *apdu.RAPDU {
	return apdu.NewRAPDU(apdu.RAPDUCommandNotAllowed)
}

func FileNotFound() *apdu.RAPDU {
	return apdu.NewRAPDU(apdu.RAPDUFileNotFound)
}

func CardKeyNotSupported() *apdu.RAPDU {
	return &apdu.RAPDU{
		SW1: 0x63,
		SW2: 0x82,
	}
}

func WrongData() *apdu.RAPDU {
	return &apdu.RAPDU{
		SW1: 0x6a,
		SW2: 0x80,
	}
}

func ReferencedDataNotFound() *apdu.RAPDU {
	return &apdu.RAPDU{
		SW1: 0x6a,
		SW2: 0x88,
	}
}

func SecurityConditionNotSatisfied() *apdu.RAPDU {
	return &apdu.RAPDU{
		SW1: 0x69,
		SW2: 0x82,
	}
}

func UnrecoverableError() *apdu.RAPDU {
	return &apdu.RAPDU{
		SW1: 0x91,
		SW2: 0xa1,
	}
}

func CommandCompleted(data []byte) *apdu.RAPDU {
	return &apdu.RAPDU{
		SW1:          0x90,
		SW2:          0x00,
		ResponseBody: data,
	}
}

func VerifyFail(retries byte) *apdu.RAPDU {
	return &apdu.RAPDU{
		SW1: 0x63,
		SW2: retries,
	}
}
