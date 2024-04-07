package networking

import (
	"fmt"
)

func HandleFrame(data []byte, connection *Connection) (bool, IChessPacket, error) {
	frame, err := frameDeserialize(data)
	if err != nil {
		// Ignore the packet. If it was malformed at the Ethernet level, it is unsalvageable
		return false, nil, fmt.Errorf("malformed ethernet frame: " + err.Error())
	}

	if frame.EtherType() != ETHER_TYPE {
		// This packet definitely wasn't intended for us
		return false, nil, fmt.Errorf("incorrect ethertype")
	}

	transport, err := ParseTransport(frame.data, frame.DestinationAddress())
	if err != nil {
		// Ignore the packet, was malformed
		return false, nil, fmt.Errorf("malformed transport: " + err.Error())
	}

	// Check how to handle this packet
	var remainingData []byte
	newConnection := false
	switch casted := transport.(type) {
	case ConnectionPacket:
		newConnection, remainingData, err = connection.Handle(casted, frame.SourceAddress())
		break
	case BroadcastPacket:
		remainingData, err = casted.handle()
		break
	}

	if err != nil {
		return newConnection, nil, err
	}

	// If there is no additional data to process, we cannot process it
	if remainingData == nil {
		return newConnection, nil, nil
	}

	// Process the additional data as a chess packet
	chessPacket, err := ChessParse(remainingData, frame.SourceAddress())

	return newConnection, chessPacket, err
}

func PackageChess(packet IChessPacket, connection *Connection) (ConnectionPacket, error) {
	// Serialize the chess packet data
	chessData, err := packet.Serialize()
	if err != nil {
		return ConnectionPacket{}, err
	}

	// Get a Connection packet containing our chess data
	transportPacket := connection.NewConnectionData(chessData)
	return transportPacket, nil
}

func PackageChessBroadcast(packet IChessPacket) ([]byte, error) {
	// Serialize the chess packet data
	chessData, err := packet.Serialize()
	if err != nil {
		return nil, err
	}

	// Make and serialize a new broadcast packet containing our chess data
	broadcastPacket := NewBroadcastPacket(chessData)
	broadcastData, err := broadcastPacket.serialize()
	if err != nil {
		return nil, err
	}

	// Make and serialize an Ethernet frame with the broadcast address as the destination
	return PackageFrame(broadcastData, GetBroadcastAddress())
}

func PackageTransport(packet ITransportPacket, connection *Connection) ([]byte, error) {
	// Serialize the transport packet data
	transportData, err := packet.serialize()
	if err != nil {
		return nil, err
	}

	// Make and serialize an Ethernet frame with the destination being our connection peer
	return PackageFrame(transportData, connection.peer)
}
