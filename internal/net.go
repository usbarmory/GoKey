// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

package gokey

import (
	"log"
	"net"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/arp"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const MTU = 1500

// Diversifier for hardware key derivation (SSH private key wrapping).
const DiversifierSSH = "GoKeySNVSOpenSSH"

func configureNetworkStack(addr tcpip.Address, nic tcpip.NICID, hwaddr string) (s *stack.Stack, link *channel.Endpoint) {
	s = stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			arp.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			icmp.NewProtocol4},
	})

	linkAddr, err := tcpip.ParseMACAddress(hwaddr)

	if err != nil {
		log.Fatal(err)
	}

	link = channel.New(256, MTU, linkAddr)
	linkEP := stack.LinkEndpoint(link)

	if err := s.CreateNIC(nic, linkEP); err != nil {
		log.Fatal(err)
	}

	if err := s.AddAddress(nic, ipv4.ProtocolNumber, addr); err != nil {
		log.Fatal(err)
	}

	subnet, err := tcpip.NewSubnet("\x00\x00\x00\x00", "\x00\x00\x00\x00")

	if err != nil {
		log.Fatal(err)
	}

	s.SetRouteTable([]tcpip.Route{{
		Destination: subnet,
		NIC:         nic,
	}})

	return
}

func startICMPEndpoint(s *stack.Stack, addr tcpip.Address, port uint16, nic tcpip.NICID) {
	var wq waiter.Queue

	fullAddr := tcpip.FullAddress{Addr: addr, Port: port, NIC: nic}
	ep, err := s.NewEndpoint(icmp.ProtocolNumber4, ipv4.ProtocolNumber, &wq)

	if err != nil {
		log.Fatalf("endpoint error (icmp): %v", err)
	}

	if err := ep.Bind(fullAddr); err != nil {
		log.Fatal("bind error (icmp endpoint): ", err)
	}
}

// StartNetworking configures and start the TCP/IP network stack.
func StartNetworking(MAC string, IP string) (s *stack.Stack, l *channel.Endpoint) {
	addr := tcpip.Address(net.ParseIP(IP)).To4()
	s, l = configureNetworkStack(addr, 1, MAC)

	// handle pings
	startICMPEndpoint(s, addr, 0, 1)

	return
}
