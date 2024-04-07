package networking

import (
	"bytes"
	"fmt"
	"net"
)

type ITransportPacket interface {
	Data() []byte
	serialize() ([]byte, error)
}

func ParseTransport(data []byte, dest net.HardwareAddr) (ITransportPacket, error) {
	// Determine whether this is a broadcast or connection packet
	var transportPacket ITransportPacket
	var err error

	if bytes.Equal(dest, GetBroadcastAddress()) {
		transportPacket, err = broadcastDeserialize(data)
	} else if bytes.Equal(dest, mac) {
		transportPacket, err = connectionDeserialize(data)
	} else {
		// Ignore, wasn't for us
		return nil, fmt.Errorf("frame not addressed to us")
	}

	if err != nil {
		// Pass the error up to be dealt with
		return nil, err
	}

	return transportPacket, nil
}
