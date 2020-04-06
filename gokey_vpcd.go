// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build vpcd

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/f-secure-foundry/GoKey/internal/icc"
)

var server string

// http://frankmorgner.github.io/vsmartcard/virtualsmartcard/api.html
const (
	POWER_OFF = 0x00
	POWER_ON  = 0x01
	RESET     = 0x02
	GET_ATR   = 0x04
)

// dummyUID for virtual smart card operation
var dummyUID = [4]byte{0xaa, 0xbb, 0xcc, 0xdd}

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	flag.StringVar(&server, "c", "127.0.0.1:35963", "vpcd address:port pair")
}

func main() {
	flag.Parse()

	// Initialize an OpenPGP card with the bundled key information (defined
	// in `keys.go` and generated at compilation time).
	card := &icc.Interface{
		Serial:     dummyUID,
		SNVS:       SNVS,
		ArmoredKey: pgpSecretKey,
		Name:       NAME,
		Language:   LANGUAGE,
		Sex:        SEX,
		URL:        URL,
		Debug:      true,
	}

	if initAtBoot {
		err := card.Init()

		if err != nil {
			log.Printf("initialization error: %v", err)
		}
	}

	for {
		conn, err := net.Dial("tcp", server)

		if err != nil {
			log.Printf("retrying (%v)", err)

			time.Sleep(1 * time.Second)
			continue
		}

		handle(conn, card)
	}
}

func handle(conn net.Conn, card *icc.Interface) {
	defer conn.Close()

	for {
		length := make([]byte, 2)
		_, err := conn.Read(length)

		if err != nil {
			log.Printf("cannot read, %v", err)
			return
		}

		res, err := handleVPCDRequest(conn, length, card)

		if err != nil {
			log.Fatalf("cannot handle request, %v", err)
		}

		_, err = conn.Write(res)

		if err != nil {
			log.Printf("cannot send response, %v", err)
			return
		}
	}
}

func handleVPCDRequest(conn net.Conn, length []byte, card *icc.Interface) (res []byte, err error) {
	if len(length) < 2 {
		err = fmt.Errorf("request too short (%d)", len(length))
		return
	}

	n := binary.BigEndian.Uint16(length[0:2])

	if n < 1 {
		err = fmt.Errorf("length too short (%d)", n)
		return
	}

	req := make([]byte, n)
	io.ReadAtLeast(conn, req, int(n))

	if card.Debug {
		log.Printf("vpcd << %x", req)
	}

	if n == 1 {
		switch req[0] {
		case POWER_OFF, POWER_ON, RESET:
			// no response
			return
		case GET_ATR:
			res = card.ATR()
		}
	} else {
		res, err = card.RawCommand(req[0:])
	}

	if err != nil {
		return
	}

	length = make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(res)))
	res = append(length, res...)

	if card.Debug {
		log.Printf("vpcd >> %x", res)
	}

	return
}
