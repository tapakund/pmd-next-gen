// SPDX-License-Identifier: Apache-2.0

package networkd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/vishvananda/netlink"

	"github.com/pm-web/pkg/configfile"
	"github.com/pm-web/pkg/web"
)

type MatchSection struct {
	Name string `json:"Name"`
}

type NetworkSection struct {
	DHCP                string   `json:"DHCP"`
	DNS                 []string `json:"DNS"`
	Domains             []string `json:"Domains"`
	NTP                 []string `json:"NTP"`
	IPv6AcceptRA        string   `json:"IPv6AcceptRA"`
	LinkLocalAddressing string   `json:"LinkLocalAddressing"`
	MulticastDNS        string   `json:"MulticastDNS"`
}

type Network struct {
	Link           string         `json:"Link"`
	MatchSection   MatchSection   `json:"MatchSection"`
	NetworkSection NetworkSection `json:"NetworkSection"`
}

type LinkState struct {
	AddressState     string   `json:"AddressState"`
	AlternativeNames []string `json:"AlternativeNames"`
	CarrierState     string   `json:"CarrierState"`
	Driver           string   `json:"Driver"`
	IPv4AddressState string   `json:"IPv4AddressState"`
	IPv6AddressState string   `json:"IPv6AddressState"`
	Index            int      `json:"Index"`
	LinkFile         string   `json:"LinkFile"`
	Model            string   `json:"Model"`
	Name             string   `json:"Name"`
	NetworkFile      string   `json:"NetworkFile"`
	OnlineState      string   `json:"OnlineState"`
	OperationalState string   `json:"OperationalState"`
	Path             string   `json:"Path"`
	SetupState       string   `json:"SetupState"`
	Type             string   `json:"Type"`
	Vendor           string   `json:"Vendor"`
}

func decodeJSONRequest(r *http.Request) (*Network, error) {
	n := Network{}
	if err := json.NewDecoder(r.Body).Decode(&n); err != nil {
		return &n, err
	}

	return &n, nil
}

func AcquireNetworkLinkProperty(ctx context.Context, w http.ResponseWriter) error {
	links, err := DBusNetworkLinkProperty(ctx)
	if err != nil {
		return err
	}

	for _, m := range links {
		fmt.Println(m)
	}

	return web.JSONResponse(links, w)
}

func (n *Network) ConfigureNetworkSection(m *configfile.Meta) {
	if n.NetworkSection.DHCP != "" {
		m.SetKeySectionString("Network", "DHCP", n.NetworkSection.DHCP)
	}

	if n.NetworkSection.IPv6AcceptRA != "" {
		m.SetKeySectionString("Network", "IPv6AcceptRA", n.NetworkSection.IPv6AcceptRA)
	}

	if n.NetworkSection.LinkLocalAddressing != "" {
		m.SetKeySectionString("Network", "LinkLocalAddressing", n.NetworkSection.LinkLocalAddressing)
	}

	if n.NetworkSection.MulticastDNS != "" {
		m.SetKeySectionString("Network", "MulticastDNS", n.NetworkSection.MulticastDNS)
	}

	if len(n.NetworkSection.Domains) > 0 {
		m.SetKeySectionString("Network", "Domains", strings.Join(n.NetworkSection.Domains, " "))
	}

	if len(n.NetworkSection.DNS) > 0 {
		m.SetKeySectionString("Network", "DNS", strings.Join(n.NetworkSection.DNS, " "))
	}

	if len(n.NetworkSection.NTP) > 0 {
		m.SetKeySectionString("Network", "NTP", strings.Join(n.NetworkSection.NTP, " "))
	}
}

func (n *Network) ConfigureNetwork(ctx context.Context, w http.ResponseWriter) error {
	link, err := netlink.LinkByName(n.Link)
	if err != nil {
		return err
	}

	network, err := CreateOrParseNetworkFile(link)
	if err != nil {
		return err
	}

	m, err := configfile.Load(network)
	if err != nil {
		return err
	}

	n.ConfigureNetworkSection(m)

	if err := m.Save(); err != nil {
		return err
	}

	if err := DBusNetworkReload(ctx); err != nil {
		return err
	}

	return web.JSONResponse("configured", w)
}
