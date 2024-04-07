package networking

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// Store our client ID
// MAC Address is not granular enough, since two clients may run on the same network interface
var clientId = uuid.New()

// Map of the last broadcast packet received from each client
var lastTimestamp = make(map[uuid.UUID]int64)

type BroadcastHeader struct {
	clientId  uuid.UUID
	timestamp int64
}

type BroadcastPacket struct {
	BroadcastHeader

	data []byte
}

func NewBroadcastPacket(data []byte) BroadcastPacket {
	packet := BroadcastPacket{
		BroadcastHeader: BroadcastHeader{
			clientId:  clientId,
			timestamp: time.Now().UnixMilli(),
		},
		data: data,
	}

	return packet
}

func (p BroadcastPacket) Data() []byte {
	return p.data
}

func (p BroadcastPacket) handle() ([]byte, error) {
	if clientId == p.clientId {
		// Don't want to process a broadcast that we sent
		return nil, fmt.Errorf("this is our own broadcast")
	}

	// Check if we have received from this address before
	lastSuccessfulTime, ok := lastTimestamp[p.clientId]
	if ok && lastSuccessfulTime == p.timestamp {
		// Have received this exact timestamp before, ignore it as a duplicate
		return nil, fmt.Errorf("duplicate broadcast received")
	}

	// This is a new packet, store the timestamp
	lastTimestamp[p.clientId] = p.timestamp
	return p.Data(), nil
}

func (p BroadcastPacket) serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// The first 16 bytes are the client UUID
	err := binary.Write(&buf, binary.BigEndian, p.clientId)
	if err != nil {
		return nil, err
	}

	// The next 4 bytes are the timestamp, for deduplication
	err = binary.Write(&buf, binary.BigEndian, p.timestamp)
	if err != nil {
		return nil, err
	}

	// Write the data as is
	buf.Write(p.data)

	return buf.Bytes(), nil
}

func broadcastDeserialize(buf []byte) (BroadcastPacket, error) {
	broadcastPacket := BroadcastPacket{}
	headerSize := binary.Size(BroadcastHeader{})

	// Read the header of the ethernet frame
	if len(buf) < headerSize {
		return BroadcastPacket{}, fmt.Errorf("error reading broadcast packet header: not long enough")
	}

	// Start reading the header
	reader := bytes.NewReader(buf[:headerSize])
	// First, read 16 bytes for the UUID of the sending machine
	err := binary.Read(reader, binary.BigEndian, &broadcastPacket.clientId)
	if err != nil {
		return BroadcastPacket{}, err
	}

	// Next, read 4 bytes for the packet timestamp
	err = binary.Read(reader, binary.BigEndian, &broadcastPacket.timestamp)
	if err != nil {
		return BroadcastPacket{}, err
	}

	// Read the rest of the data as is
	broadcastPacket.data = buf[headerSize:]

	return broadcastPacket, nil
}
