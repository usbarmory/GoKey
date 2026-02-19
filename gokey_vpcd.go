// https://github.com/usbarmory/GoKey
//
// Copyright (c) The GoKey authors. All Rights Reserved.
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build vpcd

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/usbarmory/GoKey/internal/icc"
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

	if err := card.Init(); err != nil {
		log.Printf("initialization error: %v", err)
	}

	go serveRPC(card)

	// never returns
	dialVPCD(card)
}

func serveRPC(card *icc.Interface) {
	tmp, err := os.MkdirTemp("", "")

	if err != nil {
		log.Fatalf("error creating temporary directory %v", err)
	}
	defer os.RemoveAll(tmp)

	path := filepath.Join(tmp, "p11kit.sock")

	l, err := net.Listen("unix", path)

	if err != nil {
		log.Fatalf("listening on %s: %v", path, err)
	}
	defer l.Close()

	log.Printf("export P11_KIT_SERVER_ADDRESS=unix:path=%s", path)

	for {
		conn, err := l.Accept()

		if err != nil {
			log.Printf("cannot accept, %v", err)
			continue
		}

		go func() {
			if err := card.ServeRPC(conn); err != nil {
				log.Printf("cannot handle request, %v", err)
			}
			conn.Close()
		}()
	}
}

func dialVPCD(card *icc.Interface) {
	for {
		conn, err := net.Dial("tcp", server)

		if err != nil {
			log.Printf("retrying (%v)", err)
			time.Sleep(1 * time.Second)
			continue
		}

		handleVPCDConnection(conn, card)
	}
}

func handleVPCDConnection(conn net.Conn, card *icc.Interface) {
	defer conn.Close()

	for {
		length := make([]byte, 2)

		if _, err := conn.Read(length); err != nil {
			log.Printf("cannot read, %v", err)
			return
		}

		res, err := handleVPCDRequest(conn, length, card)

		if err != nil {
			log.Fatalf("cannot handle request, %v", err)
		}

		if _, err = conn.Write(res); err != nil {
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
		if res, err = card.RawCommand(req[0:]); err != nil {
			return
		}
	}

	length = make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(res)))
	res = append(length, res...)

	if card.Debug {
		log.Printf("vpcd >> %x", res)
	}

	return
}
