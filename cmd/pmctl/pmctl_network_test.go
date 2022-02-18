// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 VMware, Inc.

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pmd-nextgen/pkg/configfile"
	"github.com/pmd-nextgen/pkg/share"
	"github.com/pmd-nextgen/pkg/system"
	"github.com/pmd-nextgen/pkg/validator"
	"github.com/pmd-nextgen/pkg/web"
	"github.com/pmd-nextgen/plugins/network/networkd"
	"github.com/pmd-nextgen/plugins/network/resolved"
	"github.com/vishvananda/netlink"
)

func setupLink(t *testing.T, link netlink.Link) {
	if err := netlink.LinkAdd(link); err != nil && err.Error() != "file exists" {
		t.Fatal(err)
	}

	if !validator.LinkExists(link.Attrs().Name) {
		t.Fatal("link does not exists")
	}
}

func removeLink(t *testing.T, link string) {
	l, err := netlink.LinkByName(link)
	if err != nil {
		t.Fatal(err)
	}

	netlink.LinkDel(l)
}

func configureNetwork(t *testing.T, n networkd.Network) (*configfile.Meta, error) {
	var resp []byte
	var err error
	resp, err = web.DispatchSocket(http.MethodPost, "", "/api/v1/network/networkd/network/configure", nil, n)
	if err != nil {
		t.Fatalf("Failed to configure network: %v\n", err)
	}

	j := web.JSONResponseMessage{}
	if err := json.Unmarshal(resp, &j); err != nil {
		t.Fatalf("Failed to decode json message: %v\n", err)
	}
	if !j.Success {
		t.Fatalf("Failed to configure network: %v\n", j.Errors)
	}

	time.Sleep(time.Second * 3)
	link, err := netlink.LinkByName("test99")
	network, err := networkd.ParseLinkNetworkFile(link.Attrs().Index)
	if err != nil {
		t.Fatalf("Failed to configure network: %v\n", err)
	}

	m, err := configfile.Load(network)
	defer os.Remove(m.Path)

	return m, err
}

func TestNetworkAddGlobalDns(t *testing.T) {
	s := []string{"8.8.8.8", "8.8.4.4", "8.8.8.1", "8.8.8.2"}
	n := resolved.GlobalDns{
		DnsServers: s,
	}
	var resp []byte
	var err error
	resp, err = web.DispatchSocket(http.MethodPost, "", "/api/v1/network/resolved/add", nil, n)
	if err != nil {
		t.Fatalf("Failed to add global Dns server: %v\n", err)
	}

	j := web.JSONResponseMessage{}
	if err := json.Unmarshal(resp, &j); err != nil {
		t.Fatalf("Failed to decode json message: %v\n", err)
	}
	if !j.Success {
		t.Fatalf("Failed to add Dns servers: %v\n", j.Errors)
	}

	time.Sleep(time.Second * 3)

	m, err := configfile.Load("/etc/systemd/resolved.conf")
	if err != nil {
		t.Fatalf("Failed to load resolved.conf: %v\n", err)
	}

	dns := m.GetKeySectionString("Resolve", "DNS")
	for _, d := range s {
		if !share.StringContains(strings.Split(dns, " "), d) {
			t.Fatalf("Failed")
		}
	}
}

func TestNetworkRemoveGlobalDns(t *testing.T) {
	TestNetworkAddGlobalDns(t)
	s := []string{"8.8.8.8", "8.8.4.4"}
	n := resolved.GlobalDns{
		DnsServers: s,
	}
	var resp []byte
	var err error
	resp, err = web.DispatchSocket(http.MethodDelete, "", "/api/v1/network/resolved/remove", nil, n)
	if err != nil {
		t.Fatalf("Failed to add global Dns servers: %v\n", err)
	}

	j := web.JSONResponseMessage{}
	if err := json.Unmarshal(resp, &j); err != nil {
		t.Fatalf("Failed to decode json message: %v\n", err)
	}
	if !j.Success {
		t.Fatalf("Failed to configure Dns: %v\n", j.Errors)
	}

	time.Sleep(time.Second * 3)

	m, err := configfile.Load("/etc/systemd/resolved.conf")
	if err != nil {
		t.Fatalf("Failed to load resolved.conf: %v\n", err)
	}

	dns := m.GetKeySectionString("Resolve", "DNS")
	for _, d := range s {
		if share.StringContains(strings.Split(dns, " "), d) {
			t.Fatalf("Failed")
		}
	}
}

func TestNetworkAddGlobalDomain(t *testing.T) {
	s := []string{"test1.com", "test2.com", "test3.com", "test4.com"}
	n := resolved.GlobalDns{
		Domains: s,
	}
	var resp []byte
	var err error
	resp, err = web.DispatchSocket(http.MethodPost, "", "/api/v1/network/resolved/add", nil, n)
	if err != nil {
		t.Fatalf("Failed to add global domain: %v\n", err)
	}

	j := web.JSONResponseMessage{}
	if err := json.Unmarshal(resp, &j); err != nil {
		t.Fatalf("Failed to decode json message: %v\n", err)
	}
	if !j.Success {
		t.Fatalf("Failed to configure domain: %v\n", j.Errors)
	}

	time.Sleep(time.Second * 3)

	m, err := configfile.Load("/etc/systemd/resolved.conf")
	if err != nil {
		t.Fatalf("Failed to load resolved.conf: %v\n", err)
	}

	domains := m.GetKeySectionString("Resolve", "Domains")
	for _, d := range s {
		if !share.StringContains(strings.Split(domains, " "), d) {
			t.Fatalf("Failed")
		}
	}
}

func TestNetworkRemoveGlobalDomain(t *testing.T) {
	TestNetworkAddGlobalDomain(t)
	s := []string{"test1.com", "test2.com"}
	n := resolved.GlobalDns{
		Domains: s,
	}
	var resp []byte
	var err error
	resp, err = web.DispatchSocket(http.MethodDelete, "", "/api/v1/network/resolved/remove", nil, n)
	if err != nil {
		t.Fatalf("Failed to add global domain: %v\n", err)
	}

	j := web.JSONResponseMessage{}
	if err := json.Unmarshal(resp, &j); err != nil {
		t.Fatalf("Failed to decode json message: %v\n", err)
	}
	if !j.Success {
		t.Fatalf("Failed to remove domain: %v\n", j.Errors)
	}

	time.Sleep(time.Second * 3)

	m, err := configfile.Load("/etc/systemd/resolved.conf")
	if err != nil {
		t.Fatalf("Failed to load resolved.conf: %v\n", err)
	}

	domains := m.GetKeySectionString("Resolve", "Domains")
	for _, d := range s {
		if share.StringContains(strings.Split(domains, " "), d) {
			t.Fatalf("Failed")
		}
	}
}

func TestNetworkAddLinkDomain(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	s := []string{"test1.com", "test2.com", "test3.com", "test4.com"}
	n := networkd.Network{
		Link: "test99",
		NetworkSection: networkd.NetworkSection{
			Domains: s,
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure link domain: %v\n", err)
	}

	domains := m.GetKeySectionString("Network", "Domains")
	for _, d := range s {
		if !share.StringContains(strings.Split(domains, " "), d) {
			t.Fatalf("Failed")
		}
	}
}

func TestNetworkRemoveLinkDomain(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	s := []string{"test1.com", "test2.com", "test3.com", "test4.com"}
	n := networkd.Network{
		Link: "test99",
		NetworkSection: networkd.NetworkSection{
			Domains: s,
		},
	}

	var resp []byte
	var err error
	resp, err = web.DispatchSocket(http.MethodPost, "", "/api/v1/network/networkd/network/configure", nil, n)
	if err != nil {
		t.Fatalf("Failed to add link domain: %v\n", err)
	}

	j := web.JSONResponseMessage{}
	if err := json.Unmarshal(resp, &j); err != nil {
		t.Fatalf("Failed to decode json message: %v\n", err)
	}
	if !j.Success {
		t.Fatalf("Failed to configure link: %v\n", j.Errors)
	}

	s = []string{"test3.com", "test4.com"}
	n = networkd.Network{
		Link: "test99",
		NetworkSection: networkd.NetworkSection{
			Domains: s,
		},
	}

	resp, err = web.DispatchSocket(http.MethodDelete, "", "/api/v1/network/networkd/network/remove", nil, n)
	if err != nil {
		t.Fatalf("Failed to remove link domain: %v\n", err)
	}

	j = web.JSONResponseMessage{}
	if err := json.Unmarshal(resp, &j); err != nil {
		t.Fatalf("Failed to decode json message: %v\n", err)
	}
	if !j.Success {
		t.Fatalf("Failed to remove domain: %v\n", j.Errors)
	}

	time.Sleep(time.Second * 3)
	link, err := netlink.LinkByName("test99")
	network, err := networkd.ParseLinkNetworkFile(link.Attrs().Index)
	if err != nil {
		t.Fatalf("Failed to configure link domain: %v\n", err)
	}

	m, err := configfile.Load(network)
	if err != nil {
		t.Fatalf("Failed to configure link domain: %v\n", err)
	}
	defer os.Remove(m.Path)

	domains := m.GetKeySectionString("Network", "Domains")
	for _, d := range s {
		if share.StringContains(strings.Split(domains, " "), d) {
			t.Fatalf("Failed")
		}
	}
}

func TestNetworkDHCP(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		NetworkSection: networkd.NetworkSection{
			DHCP: "ipv4",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure DHCP: %v\n", err)
	}

	if m.GetKeySectionString("Network", "DHCP") != "ipv4" {
		t.Fatalf("Failed to set DHCP")
	}
}

func TestNetworkLinkLocalAddressing(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		NetworkSection: networkd.NetworkSection{
			LinkLocalAddressing: "ipv4",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure LinkLocalAddressing: %v\n", err)
	}

	if m.GetKeySectionString("Network", "LinkLocalAddressing") != "ipv4" {
		t.Fatalf("Failed to set LinkLocalAddressing")
	}
}

func TestNetworkMulticastDNS(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		NetworkSection: networkd.NetworkSection{
			MulticastDNS: "resolve",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure MulticastDNS: %v\n", err)
	}

	if m.GetKeySectionString("Network", "MulticastDNS") != "resolve" {
		t.Fatalf("Failed to set MulticastDNS")
	}
}

func TestNetworkIPv6AcceptRA(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		NetworkSection: networkd.NetworkSection{
			IPv6AcceptRA: "no",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure IPv6AcceptRA: %v\n", err)
	}

	if m.GetKeySectionString("Network", "IPv6AcceptRA") != "no" {
		t.Fatalf("Failed to set IPv6AcceptRA")
	}
}

func TestNetworkDHCP4ClientIdentifier(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		DHCPv4Section: networkd.DHCPv4Section{
			ClientIdentifier: "duid",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure DHCP4ClientIdentifier: %v\n", err)
	}

	if m.GetKeySectionString("DHCPv4", "ClientIdentifier") != "duid" {
		t.Fatalf("Failed to set ClientIdentifier")
	}
}

func TestNetworkDHCPIAID(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		DHCPv4Section: networkd.DHCPv4Section{
			IAID: "8765434",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure DHCPIAID: %v\n", err)
	}

	if m.GetKeySectionString("DHCPv4", "IAID") != "8765434" {
		t.Fatalf("Failed to set IAID")
	}
}

func TestNetworkRoute(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		RouteSections: []networkd.RouteSection{
			{
				Gateway:         "192.168.0.1",
				GatewayOnlink:   "no",
				Source:          "192.168.1.15/24",
				Destination:     "192.168.10.10/24",
				PreferredSource: "192.168.8.9",
				Table:           "1234",
				Scope:           "link",
			},
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure Route: %v\n", err)
	}

	if m.GetKeySectionString("Route", "Gateway") != "192.168.0.1" {
		t.Fatalf("Failed to set Gateway")
	}
	if m.GetKeySectionString("Route", "GatewayOnlink") != "no" {
		t.Fatalf("Failed to set GatewayOnlink")
	}
	if m.GetKeySectionString("Route", "Source") != "192.168.1.15/24" {
		t.Fatalf("Failed to set Source")
	}
	if m.GetKeySectionString("Route", "Destination") != "192.168.10.10/24" {
		t.Fatalf("Failed to set Destination")
	}
	if m.GetKeySectionString("Route", "PreferredSource") != "192.168.8.9" {
		t.Fatalf("Failed to set PreferredSource")
	}
	if m.GetKeySectionString("Route", "Table") != "1234" {
		t.Fatalf("Failed to set Table")
	}
	if m.GetKeySectionString("Route", "Scope") != "link" {
		t.Fatalf("Failed to set Scope")
	}
}

func TestNetworkAddress(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		AddressSections: []networkd.AddressSection{
			{
				Address: "192.168.1.15/24",
				Peer:    "192.168.10.10/24",
				Label:   "ipv4",
				Scope:   "link",
			},
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure Route: %v\n", err)
	}

	if m.GetKeySectionString("Address", "Address") != "192.168.1.15/24" {
		t.Fatalf("Failed to set Address")
	}
	if m.GetKeySectionString("Address", "Peer") != "192.168.10.10/24" {
		t.Fatalf("Failed to set Peer")
	}
	if m.GetKeySectionString("Address", "Label") != "ipv4" {
		t.Fatalf("Failed to set Label")
	}
	if m.GetKeySectionString("Address", "Scope") != "link" {
		t.Fatalf("Failed to set Scope")
	}
}

func TestNetworkLinkMode(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		LinkSection: networkd.LinkSection{
			ARP:               "yes",
			Multicast:         "yes",
			AllMulticast:      "no",
			Promiscuous:       "no",
			RequiredForOnline: "yes",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure Link Mode: %v\n", err)
	}

	if m.GetKeySectionString("Link", "ARP") != "yes" {
		t.Fatalf("Failed to set ARP")
	}
	if m.GetKeySectionString("Link", "Multicast") != "yes" {
		t.Fatalf("Failed to set Multicast")
	}
	if m.GetKeySectionString("Link", "AllMulticast") != "no" {
		t.Fatalf("Failed to set AllMulticast")
	}
	if m.GetKeySectionString("Link", "Promiscuous") != "no" {
		t.Fatalf("Failed to set Promiscuous")
	}
	if m.GetKeySectionString("Link", "RequiredForOnline") != "yes" {
		t.Fatalf("Failed to set RequiredForOnline")
	}
}

func TestNetworkLinkMTU(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		LinkSection: networkd.LinkSection{
			MTUBytes: "2048",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure Link MTU: %v\n", err)
	}

	if m.GetKeySectionString("Link", "MTUBytes") != "2048" {
		t.Fatalf("Failed to set MTUBytes")
	}
}

func TestNetworkLinkMAC(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		LinkSection: networkd.LinkSection{
			MACAddress: "00:a0:de:63:7a:e6",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure Link MAC: %v\n", err)
	}

	if m.GetKeySectionString("Link", "MACAddress") != "00:a0:de:63:7a:e6" {
		t.Fatalf("Failed to set MACAddress")
	}
}

func TestNetworkLinkGroup(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		LinkSection: networkd.LinkSection{
			Group: "2147483647",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure Link Group: %v\n", err)
	}

	if m.GetKeySectionString("Link", "Group") != "2147483647" {
		t.Fatalf("Failed to set Group")
	}
}

func TestNetworkLinkOnlineFamily(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		LinkSection: networkd.LinkSection{
			RequiredFamilyForOnline: "ipv4",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure Link OnlineFamily: %v\n", err)
	}

	if m.GetKeySectionString("Link", "RequiredFamilyForOnline") != "ipv4" {
		t.Fatalf("Failed to set RequiredFamilyForOnline")
	}
}

func TestNetworkLinkActPolicy(t *testing.T) {
	setupLink(t, &netlink.Dummy{netlink.LinkAttrs{Name: "test99"}})
	defer removeLink(t, "test99")

	system.ExecRun("systemctl", "restart", "systemd-networkd")
	time.Sleep(time.Second * 3)

	n := networkd.Network{
		Link: "test99",
		LinkSection: networkd.LinkSection{
			ActivationPolicy: "always-up",
		},
	}

	m, err := configureNetwork(t, n)
	if err != nil {
		t.Fatalf("Failed to configure Link ActPolicy: %v\n", err)
	}

	if m.GetKeySectionString("Link", "ActivationPolicy") != "always-up" {
		t.Fatalf("Failed to set ActivationPolicy")
	}
}
