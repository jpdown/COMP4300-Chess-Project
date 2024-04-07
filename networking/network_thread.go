package networking

import (
	"net"
	"project-go/logging"
	"syscall"
)

var mac net.HardwareAddr

var SendChan chan []byte

func SendThread() {
	// Create a raw socket that can send Ethernet frames raw
	fd, _ := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, syscall.ETH_P_ALL)
	defer syscall.Close(fd)

	// Get the network interface we will be sending on
	iface, err := GetIface()
	if err != nil {
		logging.Log("Error getting default interface, will not send frames. " + err.Error())
		return
	}

	// Build a sockaddr containing our interface and MAC address
	// Linux uses this sockaddr to determine which interface to send on
	var sockaddr syscall.SockaddrLinklayer
	sockaddr.Ifindex = iface.Index
	mac = iface.HardwareAddr

	// While we are still receiving data to send, send the data as is
	// We are assuming we are given properly structured data
	for data := range SendChan {
		logging.Debugf("sending %x\n", data)
		err = syscall.Sendto(fd, data, 0, &sockaddr)
		if err != nil {
			logging.Log("Error sending data: " + err.Error())
		}
	}
}

func RecvThread(result chan<- []byte) {
	// Create a socket that will receive every raw ethernet frame
	fd, _ := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, htons(syscall.ETH_P_ALL))
	defer syscall.Close(fd)

	// Get the network interface that we will receive from
	iface, err := GetIface()
	if err != nil {
		logging.Log("Error getting default interface, will not receive frames. " + err.Error())
		return
	}

	// Bind to the network interface so we can start receiving frames
	logging.Log("Binding to interface " + iface.Name)
	err = syscall.BindToDevice(fd, iface.Name)
	if err != nil {
		logging.Log("Error binding to interface, will not receive frames. " + err.Error())
		return
	}

	// Make a buffer to receive frames into
	buf := make([]byte, 2048)

	// Read all packets in a loop, forever until the program is terminated
	for true {
		// Receive a frame into buf
		dataLen, _, err := syscall.Recvfrom(fd, buf, 0)
		if err != nil {
			logging.Log("Error reading bytes, exiting: " + err.Error())
			return
		}

		// We have received data, pass it up
		// We only have valid data in the first dataLen bytes
		result <- buf[:dataLen]
	}
}
