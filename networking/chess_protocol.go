package networking

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"project-go/chess"
	"project-go/logging"
)

type PacketType int

const (
	LOBBY_CREATED PacketType = iota
	LOBBY_LIST_REQUEST
	LOBBY_INFO
	LOBBY_JOIN_REQUEST
	LOBBY_START_REQUEST
	LOBBY_START_ACCEPT
	MOVE_PIECE
	FORFEIT
)

type IChessPacket interface {
	Source() net.HardwareAddr
	Type() PacketType
	Serialize() ([]byte, error)
}

type ChessPacket struct {
	SourceAddress net.HardwareAddr
	packetType    PacketType
}

func GetBroadcastAddress() net.HardwareAddr {
	return []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
}

func ChessParse(data []byte, source net.HardwareAddr) (IChessPacket, error) {
	reader := bytes.NewReader(data)

	// The first int32 in the packet is the packet type
	var i32Type int32
	err := binary.Read(reader, binary.BigEndian, &i32Type)
	if err != nil {
		return nil, err
	}

	logging.Debugf("received chess packet of type %d\n", i32Type)

	// The packet type tells us how we need to parse it
	pType := PacketType(i32Type)
	switch pType {
	case LOBBY_CREATED:
		return DeserializeLobbyCreatedPacket(reader, source)
	case LOBBY_LIST_REQUEST:
		return DeserializeLobbyListRequest(reader, source)
	case LOBBY_INFO:
		return DeserializeLobbyInfoPacket(reader, source)
	case LOBBY_JOIN_REQUEST:
		return DeserializeLobbyJoinRequest(reader, source)
	case LOBBY_START_REQUEST:
		return DeserializeLobbyStartRequest(reader, source)
	case LOBBY_START_ACCEPT:
		return DeserializeLobbyStartAccepted(reader, source)
	case MOVE_PIECE:
		return DeserializeMovePiecePacket(reader, source)
	case FORFEIT:
		return DeserializeForfeitPacket(reader, source)
	default:
		return nil, fmt.Errorf("invalid packet type %d", pType)
	}
}

func (c ChessPacket) Source() net.HardwareAddr {
	return c.SourceAddress
}

func (c ChessPacket) Type() PacketType {
	return c.packetType
}

type LobbyCreatedPacket struct {
	ChessPacket
	Name string
}

func NewLobbyCreated(name string) LobbyCreatedPacket {
	return LobbyCreatedPacket{
		ChessPacket: ChessPacket{
			SourceAddress: nil,
			packetType:    LOBBY_CREATED,
		},
		Name: name,
	}
}

func (p LobbyCreatedPacket) Serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// Write the type, 4 bytes
	err := binary.Write(&buf, binary.BigEndian, int32(p.Type()))
	if err != nil {
		return nil, err
	}

	// Write the name length, 4 bytes
	err = binary.Write(&buf, binary.BigEndian, int32(len(p.Name)))
	if err != nil {
		return nil, err
	}

	// Write the name, variable length
	for i := range p.Name {
		err = binary.Write(&buf, binary.BigEndian, p.Name[i])
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func DeserializeLobbyCreatedPacket(reader io.Reader, source net.HardwareAddr) (LobbyCreatedPacket, error) {
	packet := LobbyCreatedPacket{}
	packet.packetType = LOBBY_CREATED
	packet.SourceAddress = source

	// The first 4 bytes remaining are the length of the following string
	var nameLength int32
	err := binary.Read(reader, binary.BigEndian, &nameLength)
	if err != nil {
		return LobbyCreatedPacket{}, err
	}

	// We have to make a fixed length buffer of bytes to read the name into
	nameBuf := make([]byte, nameLength)
	for i := int32(0); i < nameLength; i++ {
		err = binary.Read(reader, binary.BigEndian, &nameBuf[i])
		if err != nil {
			return LobbyCreatedPacket{}, err
		}
	}

	// Now that we've read the bytes, parse it as a string
	packet.Name = string(nameBuf)

	return packet, nil
}

type LobbyListRequest struct {
	ChessPacket
}

func NewLobbyListRequest() LobbyListRequest {
	return LobbyListRequest{ChessPacket{
		SourceAddress: nil,
		packetType:    LOBBY_LIST_REQUEST,
	}}
}

func (p LobbyListRequest) Serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// Write the type, 4 bytes
	err := binary.Write(&buf, binary.BigEndian, int32(p.Type()))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DeserializeLobbyListRequest(reader io.Reader, source net.HardwareAddr) (LobbyListRequest, error) {
	packet := LobbyListRequest{}
	packet.packetType = LOBBY_LIST_REQUEST
	packet.SourceAddress = source

	// There is no body in this packet, it is purely a signal

	return packet, nil
}

type LobbyInfoPacket struct {
	ChessPacket
	Name string
}

func NewLobbyInfo(name string) LobbyInfoPacket {
	return LobbyInfoPacket{
		ChessPacket: ChessPacket{
			SourceAddress: nil,
			packetType:    LOBBY_INFO,
		},
		Name: name,
	}
}

func (p LobbyInfoPacket) Serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// Write the type, 4 bytes
	err := binary.Write(&buf, binary.BigEndian, int32(p.Type()))
	if err != nil {
		return nil, err
	}

	// Write the name length, 4 bytes
	err = binary.Write(&buf, binary.BigEndian, int32(len(p.Name)))
	if err != nil {
		return nil, err
	}

	// Write the name, variable length
	for i := range p.Name {
		err = binary.Write(&buf, binary.BigEndian, p.Name[i])
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func DeserializeLobbyInfoPacket(reader io.Reader, source net.HardwareAddr) (LobbyInfoPacket, error) {
	packet := LobbyInfoPacket{}
	packet.packetType = LOBBY_INFO
	packet.SourceAddress = source

	// The first 4 bytes are the length of the following string
	var nameLength int32
	err := binary.Read(reader, binary.BigEndian, &nameLength)
	if err != nil {
		return LobbyInfoPacket{}, err
	}

	// We have to make a fixed length buffer to read the string bytes into
	nameBuf := make([]byte, nameLength)
	for i := int32(0); i < nameLength; i++ {
		err = binary.Read(reader, binary.BigEndian, &nameBuf[i])
		if err != nil {
			return LobbyInfoPacket{}, err
		}
	}

	// Now we can parse the bytes as a string
	packet.Name = string(nameBuf)

	return packet, nil
}

type LobbyJoinRequest struct {
	ChessPacket
	Name string
}

func NewLobbyJoinRequest(name string) LobbyJoinRequest {
	return LobbyJoinRequest{
		ChessPacket: ChessPacket{
			SourceAddress: nil,
			packetType:    LOBBY_JOIN_REQUEST,
		},
		Name: name,
	}
}

func (p LobbyJoinRequest) Serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// Write the type, 4 bytes
	err := binary.Write(&buf, binary.BigEndian, int32(p.Type()))
	if err != nil {
		return nil, err
	}

	// Write the name length, 4 bytes
	err = binary.Write(&buf, binary.BigEndian, int32(len(p.Name)))
	if err != nil {
		return nil, err
	}

	// Write the name, variable length
	for i := range p.Name {
		err = binary.Write(&buf, binary.BigEndian, p.Name[i])
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func DeserializeLobbyJoinRequest(reader io.Reader, source net.HardwareAddr) (LobbyJoinRequest, error) {
	packet := LobbyJoinRequest{}
	packet.packetType = LOBBY_JOIN_REQUEST
	packet.SourceAddress = source

	// The first 4 bytes are the length of the following string
	var nameLength int32
	err := binary.Read(reader, binary.BigEndian, &nameLength)
	if err != nil {
		return LobbyJoinRequest{}, err
	}

	// We have to make a fixed length buffer of bytes to read the name into
	nameBuf := make([]byte, nameLength)
	for i := int32(0); i < nameLength; i++ {
		err = binary.Read(reader, binary.BigEndian, &nameBuf[i])
		if err != nil {
			return LobbyJoinRequest{}, err
		}
	}

	// Parse the bytes as a string
	packet.Name = string(nameBuf)

	return packet, nil
}

type LobbyStartRequest struct {
	ChessPacket
}

func NewLobbyStartRequest() LobbyStartRequest {
	return LobbyStartRequest{ChessPacket{
		SourceAddress: nil,
		packetType:    LOBBY_START_REQUEST,
	}}
}

func (p LobbyStartRequest) Serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// Write the type, 4 bytes
	err := binary.Write(&buf, binary.BigEndian, int32(p.Type()))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DeserializeLobbyStartRequest(reader io.Reader, source net.HardwareAddr) (LobbyStartRequest, error) {
	packet := LobbyStartRequest{}
	packet.packetType = LOBBY_START_REQUEST
	packet.SourceAddress = source

	// There is no body in this packet, it is purely a signal

	return packet, nil
}

type LobbyStartAccepted struct {
	ChessPacket
}

func NewLobbyStartAccepted() LobbyStartAccepted {
	return LobbyStartAccepted{ChessPacket{
		SourceAddress: nil,
		packetType:    LOBBY_START_ACCEPT,
	}}
}

func (p LobbyStartAccepted) Serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// Write the type, 4 bytes
	err := binary.Write(&buf, binary.BigEndian, int32(p.Type()))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DeserializeLobbyStartAccepted(reader io.Reader, source net.HardwareAddr) (LobbyStartAccepted, error) {
	packet := LobbyStartAccepted{}
	packet.packetType = LOBBY_START_ACCEPT
	packet.SourceAddress = source

	// There is no body in this packet, it is purely a signal

	return packet, nil
}

type MovePiecePacket struct {
	ChessPacket
	SrcPos  chess.Position
	DestPos chess.Position
}

func NewMovePiece(srcPos chess.Position, destPos chess.Position) MovePiecePacket {
	return MovePiecePacket{
		ChessPacket: ChessPacket{
			SourceAddress: nil,
			packetType:    MOVE_PIECE,
		},
		SrcPos:  srcPos,
		DestPos: destPos,
	}
}

func (p MovePiecePacket) Serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// Write the type. 4 bytes
	err := binary.Write(&buf, binary.BigEndian, int32(p.Type()))
	if err != nil {
		return nil, err
	}

	// Write the source position, 2 values at 4 bytes each
	err = binary.Write(&buf, binary.BigEndian, int32(p.SrcPos.X))
	err = binary.Write(&buf, binary.BigEndian, int32(p.SrcPos.Y))
	if err != nil {
		return nil, err
	}

	// Write the destination position, 2 values at 4 bytes each
	err = binary.Write(&buf, binary.BigEndian, int32(p.DestPos.X))
	err = binary.Write(&buf, binary.BigEndian, int32(p.DestPos.Y))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DeserializeMovePiecePacket(reader io.Reader, source net.HardwareAddr) (MovePiecePacket, error) {
	packet := MovePiecePacket{}
	packet.packetType = MOVE_PIECE
	packet.SourceAddress = source

	// Declare variables to read into
	var srcX int32
	var srcY int32
	var destX int32
	var destY int32

	// Read the two source values, 4 bytes each
	err := binary.Read(reader, binary.BigEndian, &srcX)
	if err != nil {
		return MovePiecePacket{}, err
	}

	err = binary.Read(reader, binary.BigEndian, &srcY)
	if err != nil {
		return MovePiecePacket{}, err
	}

	// Make a Position object and store in the packet object
	packet.SrcPos = chess.Position{X: int(srcX), Y: int(srcY)}

	// Read the two destination values, 4 bytes each
	err = binary.Read(reader, binary.BigEndian, &destX)
	if err != nil {
		return MovePiecePacket{}, err
	}

	err = binary.Read(reader, binary.BigEndian, &destY)
	if err != nil {
		return MovePiecePacket{}, err
	}

	// Make a position object and store in the packet object
	packet.DestPos = chess.Position{X: int(destX), Y: int(destY)}

	return packet, nil
}

type ForfeitPacket struct {
	ChessPacket
}

func NewForfeit() ForfeitPacket {
	return ForfeitPacket{ChessPacket{
		SourceAddress: nil,
		packetType:    FORFEIT,
	}}
}

func (p ForfeitPacket) Serialize() ([]byte, error) {
	buf := bytes.Buffer{}

	// Write the type, 4 bytes
	err := binary.Write(&buf, binary.BigEndian, int32(p.Type()))
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DeserializeForfeitPacket(reader io.Reader, source net.HardwareAddr) (ForfeitPacket, error) {
	packet := ForfeitPacket{}
	packet.packetType = FORFEIT
	packet.SourceAddress = source

	// There is no body in this packet, it is purely a signal

	return packet, nil
}
