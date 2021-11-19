package networkd

import (
	"path"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"

	"github.com/pm-web/pkg/configfile"
	"github.com/pm-web/pkg/system"
)

func ParseLinkString(ifindex int, key string) (string, error) {
	path := "/run/systemd/netif/links/" + strconv.Itoa(ifindex)
	v, err := configfile.ParseKeyFromSectionString(path, "", key)
	if err != nil {
		return "", err
	}

	return v, nil
}

func ParseLinkSetupState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "ADMIN_STATE")
}

func ParseLinkNetworkFile(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "NETWORK_FILE")
}

func ParseLinkOperationalState(ifindex int) (string, error) {
	return ParseLinkString(ifindex, "OPER_STATE")
}

func CreateNetworkFile(link string) (string, error) {
	file := "10-" + link + ".network"
	match := "[Match]\nName=" + link + "\n"

	if err := system.WriteFullFile(path.Join("/etc/systemd/network", file), strings.Fields(match)); err != nil {
		return "", err
	}

	return path.Join("/etc/systemd/network", file), nil

}

func CreateOrParseNetworkFile(link netlink.Link) (string, error) {
	var n string
	var err error

	if _, err := ParseLinkSetupState(link.Attrs().Index); err != nil {
		if n, err = CreateNetworkFile(link.Attrs().Name); err != nil {
			return "", err
		}

		return n, nil
	}

	n, err = ParseLinkNetworkFile(link.Attrs().Index)
	if err != nil {
		if n, err = CreateNetworkFile(link.Attrs().Name); err != nil {
			return "", err
		}
	}

	return n, nil
}
