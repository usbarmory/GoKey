// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.
//
// This program bundles OpenPGP secret and SSH public keys in the resulting
// GoKey firmware executable. If available the NXP Data Co-Processor (DCP) can
// be used to encrypt OpenPGP secret keys uniquely to the target USB armory Mk
// II board.
//
// To use such feature the GoKey firmware must be compiled on the same hardware
// it will then be executed on, setting the `SNVS` environment variable. The
// `mxs-dcp` module (https://github.com/usbarmory/mxs-dcp) must be
// loaded.
//
// IMPORTANT: the unique OTPMK internal key is available only when Secure Boot
// (HAB) is enabled, otherwise a Non-volatile Test Key (NVTK), identical for
// each SoC, is used. The secure operation of the DCP and SNVS, in production
// deployments, should always be paired with Secure Boot activation.

//go:build linux && ignore

package main

import (
	"crypto/aes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"syscall"
	"unsafe"

	"github.com/usbarmory/GoKey/internal/icc"
	"github.com/usbarmory/GoKey/internal/snvs"
	"github.com/usbarmory/GoKey/internal/u2f"
	"github.com/usbarmory/GoKey/internal/usb"

	"golang.org/x/sys/unix"
)

type af_alg_iv struct {
	ivlen uint32
	iv    [aes.BlockSize]byte
}

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
}

func main() {
	var err error
	var SNVS bool

	var sshPublicKey []byte
	var sshPrivateKey []byte
	var pgpSecretKey []byte
	var u2fPublicKey []byte
	var u2fPrivateKey []byte

	if os.Getenv("SNVS") != "" {
		SNVS = true

		if _, err = deriveKey("test", make([]byte, aes.BlockSize)); err != nil {
			log.Fatalf("SNVS requested but cbc-aes-dcp missing (%v)", err)
		}
	}

	if SNVS {
		log.Printf("████████████████████████████████████████████████████████████████████████████████")
		log.Printf("                                **  WARNING  **                                 ")
		log.Printf(" SNVS *enabled*, private keys will be encrypted for this specific hardware unit ")
		log.Printf("████████████████████████████████████████████████████████████████████████████████")
	} else {
		log.Printf("████████████████████████████████████████████████████████████████████████████████")
		log.Printf("                                **  WARNING  **                                 ")
		log.Printf("   SNVS *disabled*, private keys will be bundled *without* hardware encryption  ")
		log.Printf("████████████████████████████████████████████████████████████████████████████████")
	}

	if sshPublicKeyPath := os.Getenv("SSH_PUBLIC_KEY"); sshPublicKeyPath != "" {
		sshPublicKey, err = os.ReadFile(sshPublicKeyPath)

		if err != nil {
			log.Fatal(err)
		}
	}

	if os.Getenv("SNVS") == "ssh" && len(sshPublicKey) == 0 {
		log.Fatal("SSH_PUBLIC_KEY is required with SNVS=ssh")
	}

	if sshPrivateKeyPath := os.Getenv("SSH_PRIVATE_KEY"); sshPrivateKeyPath != "" {
		if SNVS {
			sshPrivateKey, err = encrypt(sshPrivateKeyPath, usb.DiversifierSSH)
		} else {
			sshPrivateKey, err = os.ReadFile(sshPrivateKeyPath)
		}

		if err != nil {
			log.Fatal(err)
		}
	}

	if pgpSecretKeyPath := os.Getenv("PGP_SECRET_KEY"); pgpSecretKeyPath != "" {
		if SNVS {
			pgpSecretKey, err = encrypt(pgpSecretKeyPath, icc.DiversifierPGP)
		} else {
			pgpSecretKey, err = os.ReadFile(pgpSecretKeyPath)
		}

		if err != nil {
			log.Fatal(err)
		}
	}

	if u2fPublicKeyPath := os.Getenv("U2F_PUBLIC_KEY"); u2fPublicKeyPath != "" {
		u2fPublicKey, err = os.ReadFile(u2fPublicKeyPath)

		if err != nil {
			log.Fatal(err)
		}
	}

	if u2fPrivateKeyPath := os.Getenv("U2F_PRIVATE_KEY"); u2fPrivateKeyPath != "" {
		if SNVS {
			u2fPrivateKey, err = encrypt(u2fPrivateKeyPath, u2f.DiversifierU2F)
		} else {
			u2fPrivateKey, err = os.ReadFile(u2fPrivateKeyPath)
		}

		if err != nil {
			log.Fatal(err)
		}
	}

	out, err := os.Create("tmp.go")

	if err != nil {
		log.Fatal(err)
	}

	out.WriteString(`
package main

func init() {
`)

	if SNVS {
		fmt.Fprint(out, "\tSNVS = true\n")
	}

	if os.Getenv("SNVS") == "ssh" {
		fmt.Fprint(out, "\tinitAtBoot = false\n")
	} else {
		fmt.Fprint(out, "\tinitAtBoot = true\n")
	}

	if len(sshPublicKey) > 0 {
		fmt.Fprintf(out, "\tsshPublicKey = []byte(%s)\n", strconv.Quote(string(sshPublicKey)))
	}

	if len(sshPrivateKey) > 0 {
		fmt.Fprintf(out, "\tsshPrivateKey = []byte(%s)\n", strconv.Quote(string(sshPrivateKey)))
	}

	if len(pgpSecretKey) > 0 {
		fmt.Fprintf(out, "\tpgpSecretKey = []byte(%s)\n", strconv.Quote(string(pgpSecretKey)))
		fmt.Fprintf(out, "\tURL = %s\n", strconv.Quote(os.Getenv("URL")))
		fmt.Fprintf(out, "\tNAME = %s\n", strconv.Quote(os.Getenv("NAME")))
		fmt.Fprintf(out, "\tLANGUAGE = %s\n", strconv.Quote(os.Getenv("LANGUAGE")))
		fmt.Fprintf(out, "\tSEX = %s\n", strconv.Quote(os.Getenv("SEX")))
	}

	if len(u2fPublicKey) > 0 {
		fmt.Fprintf(out, "\tu2fPublicKey = []byte(%s)\n", strconv.Quote(string(u2fPublicKey)))
	}

	if len(u2fPrivateKey) > 0 {
		fmt.Fprintf(out, "\tu2fPrivateKey = []byte(%s)\n", strconv.Quote(string(u2fPrivateKey)))
	}

	out.WriteString(`
}
`)
}

func encrypt(path string, diversifier string) (output []byte, err error) {
	input, err := os.ReadFile(path)

	if err != nil {
		return
	}

	// It is advised to use only deterministic input data for key
	// derivation, therefore we use the empty allocated IV before it being
	// filled.
	iv := make([]byte, aes.BlockSize)
	key, err := deriveKey(diversifier, iv)

	if err != nil {
		return
	}
	_, err = io.ReadFull(rand.Reader, iv)

	if err != nil {
		return
	}

	output, err = snvs.Encrypt(input, key, iv)

	return
}

// equivalent to PKCS#11 C_DeriveKey with CKM_AES_CBC_ENCRYPT_DATA
func deriveKey(diversifier string, iv []byte) (key []byte, err error) {
	if len(iv) != aes.BlockSize {
		return nil, errors.New("invalid IV size")
	}

	if len(diversifier) > aes.BlockSize {
		return nil, errors.New("invalid diversifier size")
	}

	fd, err := unix.Socket(unix.AF_ALG, unix.SOCK_SEQPACKET, 0)

	if err != nil {
		return
	}
	defer unix.Close(fd)

	addr := &unix.SockaddrALG{
		Type: "skcipher",
		Name: "cbc-aes-dcp",
	}

	err = unix.Bind(fd, addr)

	if err != nil {
		return
	}

	// https://github.com/golang/go/issues/31277
	// SetsockoptString does not allow empty strings
	_, _, e1 := syscall.Syscall6(syscall.SYS_SETSOCKOPT, uintptr(fd), uintptr(unix.SOL_ALG), uintptr(unix.ALG_SET_KEY), uintptr(0), uintptr(0), 0)

	if e1 != 0 {
		err = errors.New("setsockopt failed")
		return
	}

	apifd, _, _ := unix.Syscall(unix.SYS_ACCEPT, uintptr(fd), 0, 0)

	return cryptoAPI(apifd, unix.ALG_OP_ENCRYPT, iv, icc.Pad([]byte(diversifier), false))
}

func cryptoAPI(fd uintptr, mode uint32, iv []byte, input []byte) (output []byte, err error) {
	api := os.NewFile(fd, "cryptoAPI")

	cmsg := buildCmsg(mode, iv)

	output = make([]byte, len(input))
	err = syscall.Sendmsg(int(fd), input, cmsg, nil, 0)

	if err != nil {
		return
	}

	_, err = api.Read(output)

	return
}

func buildCmsg(mode uint32, iv []byte) []byte {
	cbuf := make([]byte, syscall.CmsgSpace(4)+syscall.CmsgSpace(20))

	cmsg := (*syscall.Cmsghdr)(unsafe.Pointer(&cbuf[0]))
	cmsg.Level = unix.SOL_ALG
	cmsg.Type = unix.ALG_SET_OP
	cmsg.SetLen(syscall.CmsgLen(4))

	op := (*uint32)(unsafe.Pointer(CMSG_DATA(cmsg)))
	*op = mode

	cmsg = (*syscall.Cmsghdr)(unsafe.Pointer(&cbuf[syscall.CmsgSpace(4)]))
	cmsg.Level = unix.SOL_ALG
	cmsg.Type = unix.ALG_SET_IV
	cmsg.SetLen(syscall.CmsgLen(20))

	alg_iv := (*af_alg_iv)(unsafe.Pointer(CMSG_DATA(cmsg)))
	alg_iv.ivlen = uint32(len(iv))
	copy(alg_iv.iv[:], iv)

	return cbuf
}

func CMSG_DATA(cmsg *syscall.Cmsghdr) unsafe.Pointer {
	return unsafe.Pointer(uintptr(unsafe.Pointer(cmsg)) + uintptr(syscall.SizeofCmsghdr))
}
