package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/j-keck/arping"
	"github.com/vishvananda/netlink"

	weavenet "github.com/weaveworks/weave/net"
)

// checkIface returns an error if the given interface cannot be found.
func checkIface(args []string) error {
	if len(args) != 1 {
		cmdUsage("check-iface", "<iface-name>")
	}
	ifaceName := args[0]

	if _, err := netlink.LinkByName(ifaceName); err != nil {
		return err
	}

	return nil
}

func delIface(args []string) error {
	if len(args) != 1 {
		cmdUsage("del-iface", "<iface-name>")
	}
	ifName := args[0]

	link, err := netlink.LinkByName(ifName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(link)
}

// setupIface renames the given iface and configures its ARP cache settings.
func setupIface(args []string) error {
	if len(args) != 2 {
		cmdUsage("setup-iface", "<iface-name> <new-iface-name>")
	}
	ifaceName := args[0]
	newIfName := args[1]

	return weavenet.SetupIface(ifaceName, newIfName)
}

// setupIfaceAddrs sets up addresses on an interface. It expects to be called inside the container's netns.
func setupIfaceAddrs(args []string) error {
	if len(args) < 1 {
		cmdUsage("setup-iface-addrs", "<iface-name> <with-multicast> <cidr>...")
	}
	link, err := netlink.LinkByName(args[0])
	if err != nil {
		return err
	}
	withMulticastRoute, err := strconv.ParseBool(args[1])
	if err != nil {
		return err
	}
	cidrs, err := parseCIDRs(args[2:])
	if err != nil {
		return err
	}
	return weavenet.SetupIfaceAddrs(link, withMulticastRoute, cidrs)
}

func configureARP(args []string) error {
	if len(args) != 2 {
		cmdUsage("configure-arp", "<iface-name-prefix> <root-path>")
	}
	prefix := args[0]
	rootPath := args[1]

	links, err := netlink.LinkList()
	if err != nil {
		return err
	}
	for _, link := range links {
		ifName := link.Attrs().Name
		if strings.HasPrefix(ifName, prefix) {
			weavenet.ConfigureARPCache(rootPath+"/proc", ifName)
			if addrs, err := netlink.AddrList(link, netlink.FAMILY_V4); err == nil {
				for _, addr := range addrs {
					arping.GratuitousArpOverIfaceByName(addr.IPNet.IP, ifName)
				}
			}
		}
	}

	return nil
}

// listNetDevs outputs network ifaces identified by the given indexes
// in the format of weavenet.Dev.
func listNetDevs(args []string) error {
	if len(args) == 0 {
		cmdUsage("list-netdevs", "<iface-index>[ <iface-index>]")
	}

	indexes := make(map[int]struct{})
	for _, index := range args {
		if index != "" {
			id, err := strconv.Atoi(index)
			if err != nil {
				return err
			}
			indexes[id] = struct{}{}
		}
	}

	links, err := netlink.LinkList()
	if err != nil {
		return err
	}

	var netdevs []weavenet.Dev

	for _, link := range links {
		if _, found := indexes[link.Attrs().Index]; found {
			netdev, err := weavenet.LinkToNetDev(link)
			if err != nil {
				return err
			}
			netdevs = append(netdevs, netdev)
		}
	}

	nds, err := json.Marshal(netdevs)
	if err != nil {
		return err
	}
	fmt.Println(string(nds))

	return nil
}
