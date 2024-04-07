package networking

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"project-go/logging"
	"strings"
)

func GetIface() (net.Interface, error) {
	// Empty string means we don't have a specific interface chosen
	var chosenInterface string = ""

	// Check every argument for --interface
	for argIndex := range os.Args {
		if strings.Contains(os.Args[argIndex], "--interface=") {
			// We have --interface, split on the =
			split := strings.Split(os.Args[argIndex], "=")
			// The second part is our chosen interface
			if len(split) > 1 {
				chosenInterface = split[1]
			}
		}
	}

	// Get the default interface if the user did not specify
	if chosenInterface == "" {
		return GetDefaultIface()
	}

	// Grab all of the network interfaces on the machine
	interfaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, err
	}

	// Scan each interface for the chosen name
	for i := range interfaces {
		iface := interfaces[i]
		if iface.Name == chosenInterface {
			return iface, nil
		}
	}

	// We were unable to find the interface
	logging.Log("The chosen interface does not exist. Attaching to default interface.")
	return GetDefaultIface()
}

func GetDefaultIface() (net.Interface, error) {
	// Finds interface with the machine's hostname

	// First, we grab the machines hostname
	hostname, err := os.Hostname()
	if err != nil {
		return net.Interface{}, err
	}

	// Next, we look up the IP for this hostname
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return net.Interface{}, err
	}
	if len(ips) < 1 {
		return net.Interface{}, fmt.Errorf("could not find an IP with your machine's hostname")
	}

	// Grab every interface on the machine
	interfaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, err
	}

	var foundIface net.Interface
	found := false

	// Loop through every interface until we find an interface with the hostname's IP
	for i := 0; i < len(interfaces) && !found; i++ {
		iface := interfaces[i]

		// Interfaces can have multiple addresses, usually for IPv4 and IPv6
		addrs, err := iface.Addrs()
		if err != nil {
			return net.Interface{}, err
		}

		// Check each interface address for if it matches the first IP we found from the hostname
		for _, addr := range addrs {
			// Go returns the addresses in an inconsistent way, this wrangles them into the same format
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				logging.Debugf("error parsing cidr " + err.Error())
				continue
			}

			if ip.Equal(ips[0]) {
				// It matches the first IP from the hostname, this is probably the default interface
				foundIface = iface
				found = true
				break
			}
		}
	}

	// We can't do much if we can't find an interface, so just return an error
	if !found {
		return net.Interface{}, fmt.Errorf("error finding default network interface")
	}

	return foundIface, nil
}

func htons(value uint16) int {
	// Reverses the byte order of a 2 byte value, from little endian to big endian
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, value)
	return int(binary.BigEndian.Uint16(buf))
}
