// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package gokey

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"

	"github.com/f-secure-foundry/GoKey/internal/icc"
	"github.com/f-secure-foundry/GoKey/internal/snvs"
	"github.com/f-secure-foundry/GoKey/internal/u2f"

	"github.com/f-secure-foundry/tamago/soc/imx6"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

const help = `
  help                          # this help
  exit, quit                    # close session
  rand                          # gather 32 bytes from TRNG via crypto/rand
  reboot                        # restart

  init                          # initialize OpenPGP smartcard
  status                        # display OpenPGP card status
  lock   (all|sig|dec)          # key lock
  unlock (all|sig|dec)          # key unlock, prompts decryption passphrase

  u2f                           # initialize U2F token
  p                             # confirm user presence
`

var card *icc.Interface

var lockCommandPattern = regexp.MustCompile(`(lock|unlock) (all|sig|dec)`)

func lockCommand(term *terminal.Terminal, op string, arg string) (res string) {
	var err error
	var pws []byte

	if !card.Initialized() {
		return "card not initialized, forgot to issue 'init' first?"
	}

	if arg == "sig" || arg == "all" {
		pws = append(pws, icc.PW1_CDS)
	}

	if arg == "dec" || arg == "all" {
		pws = append(pws, icc.PW1)
	}

	if len(pws) == 0 {
		return
	}

	switch op {
	case "lock":
		for _, pw := range pws {
			if _, err := card.Verify(icc.PW_LOCK, pw, nil); err != nil {
				return err.Error()
			}
		}
	case "unlock":
		var passphrase string

		passphrase, err = term.ReadPassword("Passphrase: ")

		if err != nil {
			break
		}

		for _, pw := range pws {
			_, err = card.Verify(icc.PW_VERIFY, pw, []byte(passphrase))
		}
	}

	if err != nil {
		return err.Error()
	}

	return
}

func handleCommand(term *terminal.Terminal, cmd string) (err error) {
	var res string

	switch cmd {
	case "exit", "quit":
		res = "logout"
		err = io.EOF
	case "help":
		res = string(term.Escape.Cyan) + help + string(term.Escape.Reset)
	case "init":
		err = card.Init()
	case "u2f":
		err = u2f.Init(true)
	case "p":
		select {
		case u2f.Presence <- true:
		default:
			res = "presence not requested"
		}
	case "rand":
		buf := make([]byte, 32)
		_, _ = rand.Read(buf)
		res = string(term.Escape.Cyan) + fmt.Sprintf("%x", buf) + string(term.Escape.Reset)
	case "reboot":
		reboot()
	case "status":
		res = card.Status()
	default:
		m := lockCommandPattern.FindStringSubmatch(cmd)

		if len(m) == 3 {
			res = lockCommand(term, m[1], m[2])
		} else {
			res = "unknown command, type `help`"
		}
	}

	fmt.Fprintln(term, res)

	return
}

func handleChannel(newChannel ssh.NewChannel) {
	if t := newChannel.ChannelType(); t != "session" {
		_ = newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	conn, requests, err := newChannel.Accept()

	if err != nil {
		log.Printf("error accepting channel, %v", err)
		return
	}

	term := terminal.NewTerminal(conn, "")
	term.SetPrompt(string(term.Escape.Red) + "> " + string(term.Escape.Reset))

	go func() {
		defer conn.Close()

		imx6.SetARMFreq(900)
		defer imx6.SetARMFreq(198)

		log.SetOutput(io.MultiWriter(os.Stdout, term))
		defer log.SetOutput(os.Stdout)

		fmt.Fprintf(term, "%s\n", Banner)
		fmt.Fprintf(term, "%s\n", string(term.Escape.Cyan)+help+string(term.Escape.Reset))

		for {
			cmd, err := term.ReadLine()

			if err == io.EOF {
				break
			}

			if err != nil {
				log.Printf("readline error: %v", err)
				continue
			}

			err = handleCommand(term, cmd)

			if err == io.EOF {
				break
			}

			if err != nil {
				log.Printf("error: %v", err)
			}
		}

		log.Printf("closing ssh connection")
	}()

	go func() {
		for req := range requests {
			reqSize := len(req.Payload)

			switch req.Type {
			case "shell":
				// do not accept payload commands
				if len(req.Payload) == 0 {
					_ = req.Reply(true, nil)
				}
			case "pty-req":
				// p10, 6.2.  Requesting a Pseudo-Terminal, RFC4254
				if reqSize < 4 {
					log.Printf("malformed pty-req request")
					continue
				}

				termVariableSize := int(req.Payload[3])

				if reqSize < 4+termVariableSize+8 {
					log.Printf("malformed pty-req request")
					continue
				}

				w := binary.BigEndian.Uint32(req.Payload[4+termVariableSize:])
				h := binary.BigEndian.Uint32(req.Payload[4+termVariableSize+4:])

				_ = term.SetSize(int(w), int(h))
				_ = req.Reply(true, nil)
			case "window-change":
				// p10, 6.7.  Window Dimension Change Message, RFC4254
				if reqSize < 8 {
					log.Printf("malformed window-change request")
					continue
				}

				w := binary.BigEndian.Uint32(req.Payload)
				h := binary.BigEndian.Uint32(req.Payload[4:])

				_ = term.SetSize(int(w), int(h))
			}
		}
	}()
}

func handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func startSSHServer(s *stack.Stack, addr tcpip.Address, port uint16, nic tcpip.NICID, authorizedKey []byte, privateKey []byte, started chan bool) {
	var key interface{}
	var err error

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(authorizedKey)

	if err != nil {
		log.Fatal("invalid authorized key: ", err)
	}

	fullAddr := tcpip.FullAddress{Addr: addr, Port: port, NIC: nic}
	listener, err := gonet.ListenTCP(s, fullAddr, ipv4.ProtocolNumber)

	if err != nil {
		log.Fatal("listener error: ", err)
	}

	srv := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), pubKey.Marshal()) {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	if len(privateKey) != 0 {
		key, err = ssh.ParseRawPrivateKey(privateKey)
	} else {
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}

	if err != nil {
		log.Fatal("private key error: ", err)
	}

	signer, err := ssh.NewSignerFromKey(key)

	if err != nil {
		log.Fatal("key conversion error: ", err)
	}

	log.Printf("starting ssh server (%s) at %s:%d", ssh.FingerprintSHA256(signer.PublicKey()), addr.String(), port)

	srv.AddHostKey(signer)

	started <- true

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Printf("error accepting connection, %v", err)
			continue
		}

		sshConn, chans, reqs, err := ssh.NewServerConn(conn, srv)

		if err != nil {
			log.Printf("error accepting handshake, %v", err)
			continue
		}

		log.Printf("new ssh connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

		go ssh.DiscardRequests(reqs)
		go handleChannels(chans)
	}
}

// StartSSHServer configures and start the management SSH server.
func StartSSHServer(s *stack.Stack, IP string, authorizedKey []byte, privateKey []byte, c *icc.Interface, started chan bool) (err error) {
	addr := tcpip.Address(net.ParseIP(IP)).To4()
	card = c

	if card.SNVS && len(privateKey) != 0 {
		privateKey, err = snvs.Decrypt(privateKey, []byte(DiversifierSSH))

		if err != nil {
			return fmt.Errorf("key decryption failed, %v", err)
		}
	}

	go func() {
		startSSHServer(s, addr, 22, 1, authorizedKey, privateKey, started)
	}()

	return
}
