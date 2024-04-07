package networking

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

// "Arbitrary" ethertype that should be unused by other protocols
// Refers to the characters 9S and 2B in the game NieR:Automata
const ETHER_TYPE = 0x9528

type EthernetFrameHeader struct {
	sourceAddress      [6]byte
	destinationAddress [6]byte
	etherType          uint16
}

type EthernetFrame struct {
	EthernetFrameHeader
	data []byte
}

func (f EthernetFrame) EtherType() uint16 {
	return f.etherType
}

func (f EthernetFrame) SourceAddress() net.HardwareAddr {
	// Return a slice containing all elements
	return f.sourceAddress[:]
}

func (f EthernetFrame) DestinationAddress() net.HardwareAddr {
	// Return a slice containing all elements
	return f.destinationAddress[:]
}

func PackageFrame(data []byte, dest net.HardwareAddr) ([]byte, error) {
	frame := EthernetFrame{
		EthernetFrameHeader: EthernetFrameHeader{
			etherType: ETHER_TYPE,
		},
		data: data,
	}

	// Fill in the source and destination addresses
	// Need to copy for compatibility with older Go versions
	copy(frame.sourceAddress[:], mac)
	copy(frame.destinationAddress[:], dest)

	return frame.serialize()
}

func (f EthernetFrame) serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// First is the destination MAC address (6 bytes)
	err := binary.Write(&buf, binary.BigEndian, f.destinationAddress)
	if err != nil {
		return nil, err
	}
	// Next is the source MAC address (6 bytes)
	err = binary.Write(&buf, binary.BigEndian, f.sourceAddress)
	if err != nil {
		return nil, err
	}
	// Next is the ethertype (2 bytes)
	err = binary.Write(&buf, binary.BigEndian, f.etherType)
	if err != nil {
		return nil, err
	}

	// The rest of the ethernet frame is the data
	buf.Write(f.data)

	return buf.Bytes(), nil
}

func frameDeserialize(buf []byte) (EthernetFrame, error) {
	frame := EthernetFrame{}
	headerSize := binary.Size(EthernetFrameHeader{})

	// Ensure we have at least enough bytes for the header
	if len(buf) < headerSize {
		return EthernetFrame{}, fmt.Errorf("error reading ethernet frame header: not long enough")
	}

	reader := bytes.NewReader(buf[:headerSize])
	// First 6 bytes are the destination address
	err := binary.Read(reader, binary.BigEndian, &frame.destinationAddress)
	if err != nil {
		return EthernetFrame{}, fmt.Errorf("error reading frame destination: %s", err.Error())
	}

	// Next 6 bytes are the source address
	err = binary.Read(reader, binary.BigEndian, &frame.sourceAddress)
	if err != nil {
		return EthernetFrame{}, fmt.Errorf("error reading frame source: %s", err.Error())
	}

	// Next 2 bytes are the ethertype
	err = binary.Read(reader, binary.BigEndian, &frame.etherType)
	if err != nil {
		return EthernetFrame{}, fmt.Errorf("error reading ethertype: %s", err.Error())
	}

	// The rest of the buffer is the data of the frame
	frame.data = buf[headerSize:]

	return frame, nil
}
