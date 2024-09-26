// https://github.com/usbarmory/GoKey
//
// Copyright (c) WithSecure Corporation
// https://foundry.withsecure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

//go:build tamago && arm

package usb

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/usbarmory/GoKey/internal/age"
	"github.com/usbarmory/GoKey/internal/icc"
	"github.com/usbarmory/GoKey/internal/snvs"
	"github.com/usbarmory/GoKey/internal/u2f"

	"github.com/usbarmory/tamago/soc/nxp/imx6ul"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

const help = `
  help                          # this help
  exit, quit                    # close session
  rand                          # gather 32 bytes from TRNG via crypto/rand
  reboot                        # restart
  status                        # display smartcard/token status

  init                          # initialize OpenPGP smartcard
  lock   (all|sig|dec)          # OpenPGP key(s) lock
  unlock (all|sig|dec)          # OpenPGP key(s) unlock, prompts passphrase

  rpc                           # PKCS#11 RPC socket
                                # use with 'ssh -L p11kit.sock:127.0.0.1:22'

  age-plugin (gen|identity-v1)  # handle age plugin state machine

  u2f                           # initialize U2F token w/  user presence test
  u2f !test                     # initialize U2F token w/o user presence test
  p                             # confirm user presence
`

// Console represents the management SSH server instance.
type Console struct {
	// AuthorizedKey is the public key for SSH client authentication, it
	// can be bundled at compile time.
	AuthorizedKey []byte

	// PrivateKey is the private key for the management SSH server, it can
	// be bundled at compile time (encrypted if secure boot is present).
	//
	// If left empty it is generated at Start() either randomly (w/o secure
	// boot) or uniquely for each device (w/ secure boot).
	PrivateKey []byte

	// Card is the OpenPGP smartcard instance.
	Card *icc.Interface
	// Token is the U2F token instance.
	Token *u2f.Token
	// PLugin is the age plugin instance.
	Plugin *age.Plugin

	Started  chan bool
	Listener net.Listener
	Banner   string

	term *terminal.Terminal
}

var lockCommandPattern = regexp.MustCompile(`(lock|unlock) (all|sig|dec)`)
var pageCommandPattern = regexp.MustCompile(`age-plugin (.*)`)

func (c *Console) lockCommand(op string, arg string) (res string) {
	var err error
	var pws []byte

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
		if !c.Card.Initialized() {
			return "card not initialized"
		}

		for _, pw := range pws {
			if _, err := c.Card.Verify(icc.PW_LOCK, pw, nil); err != nil {
				return err.Error()
			}
		}
	case "unlock":
		var passphrase string

		if !c.Card.Initialized() {
			if err = c.Card.Init(); err != nil {
				break
			}
		}

		if passphrase, err = c.term.ReadPassword("Passphrase: "); err != nil {
			break
		}

		for _, pw := range pws {
			if _, err = c.Card.Verify(icc.PW_VERIFY, pw, []byte(passphrase)); err != nil {
				break
			}
		}
	}

	if err != nil {
		return err.Error()
	}

	return
}

func (c *Console) handleTerminal(conn ssh.Channel) {
	log.SetOutput(io.MultiWriter(os.Stdout, c.term))
	defer log.SetOutput(os.Stdout)

	fmt.Fprintf(c.term, "%s\n", c.Banner)
	fmt.Fprintf(c.term, "%s\n", string(c.term.Escape.Cyan)+help+string(c.term.Escape.Reset))

	for {
		cmd, err := c.term.ReadLine()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Printf("readline error: %v", err)
			continue
		}

		err = c.handleCommand(conn, cmd)

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Printf("error: %v", err)
		}
	}

	log.Printf("closing ssh connection")
	conn.Close()
}

func (c *Console) handleCommand(conn ssh.Channel, cmd string) (err error) {
	var res string

	switch cmd {
	case "exit", "quit":
		res = "logout"
		err = io.EOF
	case "help":
		res = string(c.term.Escape.Cyan) + help + string(c.term.Escape.Reset)
	case "init":
		err = c.Card.Init()
	case "rpc":
		return c.Card.ServeRPC(conn)
	case "u2f":
		c.Token.Presence = make(chan bool)
		err = c.Token.Init()
	case "u2f !test":
		c.Token.Presence = nil
		err = c.Token.Init()
	case "p":
		if !c.Token.Initialized() {
			res = "token not initialized, issue 'u2f' first"
		} else if c.Token.Presence == nil {
			res = "U2F presence not required"
		} else {
			select {
			case c.Token.Presence <- true:
			default:
				res = "U2F presence not requested"
			}
		}
	case "rand":
		buf := make([]byte, 32)
		_, _ = rand.Read(buf)
		res = string(c.term.Escape.Cyan) + fmt.Sprintf("%x", buf) + string(c.term.Escape.Reset)
	case "reboot":
		imx6ul.Reset()
	case "status":
		res = strings.Join([]string{c.Card.Status(), c.Token.Status()}, "")
	default:
		if m := pageCommandPattern.FindStringSubmatch(cmd); len(m) == 2 {
			if !c.Plugin.Initialized() {
				res = "plugin not initialized"
			} else {
				res = c.Plugin.Handle(conn, m[1])
			}
		} else if m := lockCommandPattern.FindStringSubmatch(cmd); len(m) == 3 {
			res = c.lockCommand(m[1], m[2])
		} else {
			return errors.New("unknown command, type `help`")
		}
	}

	fmt.Fprintln(c.term, res)

	return
}

// handleDirectForward forwards the `rpc` command regardless of the request
func (c *Console) handleDirectForward(srvConn *ssh.ServerConn, newChannel ssh.NewChannel) {
	conn, _, err := newChannel.Accept()

	if err != nil {
		log.Printf("error accepting channel, %v", err)
		return
	}

	c.handleCommand(conn, "rpc")
	conn.Close()
}

func (c *Console) handleSession(newChannel ssh.NewChannel) {
	conn, requests, err := newChannel.Accept()

	if err != nil {
		log.Printf("error accepting channel, %v", err)
		return
	}

	c.term = terminal.NewTerminal(conn, "")
	c.term.SetPrompt(string(c.term.Escape.Red) + "> " + string(c.term.Escape.Reset))

	go func() {
		for req := range requests {
			reqSize := len(req.Payload)

			switch req.Type {
			case "exec":
				cmd := string(req.Payload[4:])
				c.handleCommand(conn, cmd)
				conn.Close()
				return
			case "shell":
				go c.handleTerminal(conn)
				req.Reply(true, nil)
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

				_ = c.term.SetSize(int(w), int(h))
				_ = req.Reply(true, nil)
			case "window-change":
				// p10, 6.7.  Window Dimension Change Message, RFC4254
				if reqSize < 8 {
					log.Printf("malformed window-change request")
					continue
				}

				w := binary.BigEndian.Uint32(req.Payload)
				h := binary.BigEndian.Uint32(req.Payload[4:])

				_ = c.term.SetSize(int(w), int(h))
			}
		}
	}()
}

func (c *Console) handleChannel(srvConn *ssh.ServerConn, newChannel ssh.NewChannel) {
	switch newChannel.ChannelType() {
	case "direct-tcpip":
		c.handleDirectForward(srvConn, newChannel)
	case "session":
		c.handleSession(newChannel)
	default:
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", newChannel.ChannelType()))
	}
}

func (c *Console) handleChannels(srvConn *ssh.ServerConn, chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		go c.handleChannel(srvConn, newChannel)
	}
}

func (c *Console) start(key interface{}) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(c.AuthorizedKey)

	if err != nil {
		log.Fatal("invalid authorized key: ", err)
	}

	srv := &ssh.ServerConfig{
		PublicKeyCallback: func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), pubKey.Marshal()) {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", meta.User())
		},
	}

	signer, err := ssh.NewSignerFromKey(key)

	if err != nil {
		log.Fatal("key conversion error: ", err)
	}

	srv.AddHostKey(signer)

	log.Printf("starting ssh server (%s)", ssh.FingerprintSHA256(signer.PublicKey()))
	c.Started <- true

	for {
		conn, err := c.Listener.Accept()

		c.Card.Wake()

		if err != nil {
			log.Printf("error accepting connection, %v", err)
			continue
		}

		srvConn, chans, reqs, err := ssh.NewServerConn(conn, srv)

		if err != nil {
			log.Printf("error accepting handshake, %v", err)
			continue
		}

		log.Printf("new ssh connection from %s (%s)", srvConn.RemoteAddr(), srvConn.ClientVersion())

		go ssh.DiscardRequests(reqs)
		go c.handleChannels(srvConn, chans)
	}
}

// Start configures and starts the management SSH server.
func (c *Console) Start() (err error) {
	var key interface{}

	if len(c.PrivateKey) != 0 {
		if c.Card.SNVS || c.Token.SNVS {
			c.PrivateKey, _ = snvs.Decrypt(c.PrivateKey, []byte(DiversifierSSH))
		}

		key, err = ssh.ParseRawPrivateKey(c.PrivateKey)
	} else if c.Card.SNVS || c.Token.SNVS {
		key, err = snvs.DeviceKey()
	} else {
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}

	if err != nil {
		log.Fatal("private key error: ", err)
	}

	go func() {
		c.start(key)
	}()

	return
}
