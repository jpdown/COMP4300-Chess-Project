package networking

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/google/uuid"
	"net"
	"project-go/logging"
	"project-go/util"
	"time"
)

// Hardcoded window size of 4, the volume of packets is low enough that this will never get hit anyways
const WINDOW_SIZE int = 4

// Add some time on top of the round trip time for processing
const GRACE_PERIOD time.Duration = time.Millisecond * 50

// We allow 5 losses before considering the connection dead
const MAX_LOSSES = 5

// Pulled out into its own struct for easier reading
type ConnectionHeader struct {
	sourceMachine uuid.UUID
	destMachine   uuid.UUID
	sequence      uint32

	packetType ConnectionPacketType
}

type ConnectionPacket struct {
	ConnectionHeader

	data []byte
}

type ConnectionPacketType int32

const (
	CONNECTION_REQUEST ConnectionPacketType = iota
	CONNECTION_RESPONSE
	CONNECTION_DATA
	CONNECTION_ACK
	CONNECTION_CLOSE
)

type ConnectionState int

const (
	IDLE ConnectionState = iota
	REQUESTED
	RESPONDED
	ESTABLISHED
)

// Collection of everything we need to track for an open connection
type Connection struct {
	peer            net.HardwareAddr   // MAC address of peer
	destId          uuid.UUID          // The peer's machine ID, in case there are multiple clients
	state           ConnectionState    // Which state the connection is in, which changes how we parse
	sentSeq         uint32             // The last sequence number we sent
	expectedRecvSeq uint32             // The next sequence number we expect to receive
	ackSent         time.Time          // When we sent the initial packet
	ackRecv         time.Time          // When we received a response to our initial packet
	sendWindow      []ConnectionPacket // The window for which packets to send
	ackQueue        []ConnectionPacket // Acknowledgements are not subject to resending
	windowPos       int                // Where in the send window we currently are
	lossDeadline    time.Time          // Which time we can declare a packet lost and resend the window
	numLosses       int                // How many losses we've sustained, resets upon receiving an ack
}

func NewConnection() *Connection {
	conn := Connection{}
	// Set all of the default values for an empty connection
	conn.reset()
	return &conn
}

func (c *Connection) IsActive() bool {
	return c.state != IDLE
}

func (c *Connection) Peer() net.HardwareAddr {
	return c.peer
}

func (c *Connection) SetPeer(addr net.HardwareAddr) {
	c.peer = addr
}

func (c *Connection) NewConnectionRequest() ConnectionPacket {
	packet := ConnectionPacket{
		ConnectionHeader: ConnectionHeader{
			sourceMachine: clientId,
			destMachine:   uuid.UUID{},
			sequence:      c.sentSeq,
			packetType:    CONNECTION_REQUEST,
		},
		data: nil,
	}

	return packet
}

func (c *Connection) NewConnectionAck(sequence uint32) ConnectionPacket {
	packet := ConnectionPacket{
		ConnectionHeader: ConnectionHeader{
			sourceMachine: clientId,
			destMachine:   c.destId,
			sequence:      sequence,
			packetType:    CONNECTION_ACK,
		},
		data: nil,
	}

	return packet
}

func (c *Connection) NewConnectionResponse() ConnectionPacket {
	packet := ConnectionPacket{
		ConnectionHeader: ConnectionHeader{
			sourceMachine: clientId,
			destMachine:   c.destId,
			sequence:      c.sentSeq,
			packetType:    CONNECTION_RESPONSE,
		},
		data: nil,
	}

	return packet
}

func (c *Connection) NewConnectionData(data []byte) ConnectionPacket {
	connection := ConnectionPacket{
		ConnectionHeader: ConnectionHeader{
			sourceMachine: clientId,
			destMachine:   c.destId,
			sequence:      c.sentSeq,
			packetType:    CONNECTION_DATA,
		},
		data: data,
	}

	return connection
}

func (c *Connection) NewConnectionClose() ConnectionPacket {
	connection := ConnectionPacket{
		ConnectionHeader: ConnectionHeader{
			sourceMachine: clientId,
			destMachine:   c.destId,
			sequence:      c.sentSeq,
			packetType:    CONNECTION_CLOSE,
		},
		data: nil,
	}

	return connection
}

func (c *Connection) Handle(packet ConnectionPacket, source net.HardwareAddr) (bool, []byte, error) {
	var extraData []byte = nil
	var response *ConnectionPacket = nil
	var err error = nil

	logging.Debugf("received connection packet, type %d sequence %d\n", packet.packetType, packet.sequence)

	if c.state != IDLE && c.state != REQUESTED && clientId != packet.destMachine {
		logging.Debugf("not addressed to us, addressed to %x, we are %x\n", packet.destMachine, clientId)
		return false, nil, fmt.Errorf("packet not addressed to us")
	}

	if c.state != IDLE && c.state != REQUESTED && c.destId != packet.sourceMachine {
		return false, nil, fmt.Errorf("packet not received from connection peer")
	}

	if packet.packetType != CONNECTION_ACK && packet.sequence > c.expectedRecvSeq {
		logging.Debugf("ignoring out of order packet, got %d expected max of %d\n", packet.sequence, c.expectedRecvSeq+1)
		return false, nil, fmt.Errorf("packet received out of order")
	}

	// Handle each packet type
	connectionStatusChanged := false
	switch packet.packetType {
	case CONNECTION_REQUEST:
		response, err = c.handleRequest(packet, source)
		break
	case CONNECTION_RESPONSE:
		response, err = c.handleResponse(packet)
		connectionStatusChanged = true
		break
	case CONNECTION_ACK:
		connectionStatusChanged, err = c.handleAck(packet)
		break
	case CONNECTION_DATA:
		response, extraData, err = c.handleData(packet)
		if err != nil {
			return false, nil, err
		}
		break
	case CONNECTION_CLOSE:
		if c.IsActive() {
			c.reset()
			connectionStatusChanged = true
		}
		break
	}

	// Exit early if there was an error handling the packet
	if err != nil {
		return false, nil, err
	}

	// ACKs don't increment the sequence number
	if packet.packetType != CONNECTION_ACK && packet.sequence == c.expectedRecvSeq {
		c.expectedRecvSeq++
	}

	// Queue our response if we have one
	if response != nil {
		c.QueuePacket(*response)
	}

	return connectionStatusChanged, extraData, nil
}

func (c *Connection) QueuePacket(packet ConnectionPacket) {
	if packet.packetType == CONNECTION_ACK {
		// Acks are in a separate queue because they don't get resent
		c.ackQueue = append(c.ackQueue, packet)
	} else {
		// This is a new packet, so we need to increment the sequence number
		c.sendWindow = append(c.sendWindow, packet)
		c.sentSeq++
	}
}

func (c *Connection) GetPackets() []ConnectionPacket {
	// From is the minimum of: window position, window size, num packets in send queue
	from := util.Min(c.windowPos, WINDOW_SIZE)
	from = util.Min(from, len(c.sendWindow))

	// To is the minimum of window size, num packets in send queue
	to := util.Min(WINDOW_SIZE, len(c.sendWindow))

	// Get our slice of packets to send
	slice := c.sendWindow[from:to]

	if len(slice) == 0 {
		return nil
	}

	// We're sending these packets, so this is our new window position
	c.windowPos = to

	c.setDeadline()

	logging.Debugf("sending frames %d to %d total queue %d\n", slice[0].sequence, slice[len(slice)-1].sequence, len(c.sendWindow))

	return slice
}

func (c *Connection) GetAckPackets() []ConnectionPacket {
	// We just want to send all of the ack packets, no window size
	slice := c.ackQueue[:]

	if len(slice) == 0 {
		return nil
	}

	// Empty the queue
	c.ackQueue = c.ackQueue[:0]

	logging.Debugf("sending acks %d to %d\n", slice[0].sequence, slice[len(slice)-1].sequence)

	return slice
}

func (c *Connection) CheckLoss() bool {
	if !c.lossDeadline.IsZero() && time.Now().After(c.lossDeadline) {
		// We had a frame loss, reset the window
		c.windowPos = 0
		c.numLosses++

		logging.Debug("we had a loss")

		// Close the connection if we continue to get losses
		if c.numLosses > MAX_LOSSES {
			c.Close()
			return true
		}
	}

	return false
}

func (c *Connection) Open(peer net.HardwareAddr) error {
	// We can only have one active connection at a time
	if c.IsActive() {
		return fmt.Errorf("there is already an active connection")
	}

	// We need to know who to send packets to
	c.SetPeer(peer)

	// Queue the connection request
	transportPacket := c.NewConnectionRequest()
	c.QueuePacket(transportPacket)
	c.state = REQUESTED

	// Time how long we take to get a response
	c.ackSent = time.Now()

	return nil
}

func (c *Connection) Close() {
	// Forcefully send a connection closed packet, since we are about to remove all of our state tracking
	// If this frame is lost, the other party will eventually figure out that we're gone
	transportData, err := PackageTransport(c.NewConnectionClose(), c)
	if err != nil {
		logging.Debugf("error sending connection close: " + err.Error())
	}
	SendChan <- transportData

	// Remove our connection tracking
	c.reset()
}

func (c ConnectionPacket) Data() []byte {
	return c.data
}

func (c *Connection) reset() {
	// Get rid of all state from the connection
	c.state = IDLE
	c.sentSeq = 0
	c.expectedRecvSeq = 0
	c.sendWindow = make([]ConnectionPacket, 0)
	c.ackQueue = make([]ConnectionPacket, 0)
	c.windowPos = 0
	c.peer = nil
	c.numLosses = 0
	c.ackSent = time.Time{}
	c.ackRecv = time.Time{}
}

func (c *Connection) setDeadline() {
	// Update our loss deadline so that we can resend if necessary
	c.lossDeadline = time.Now().Add(GRACE_PERIOD)
	if c.state == ESTABLISHED {
		// Only do round trip time if we know what the round trip time is
		c.lossDeadline = c.lossDeadline.Add(c.ackRecv.Sub(c.ackSent))
	}
}

func (c *Connection) handleRequest(packet ConnectionPacket, source net.HardwareAddr) (*ConnectionPacket, error) {
	// If we already have a connection, we ignore this
	if c.state != IDLE {
		return nil, fmt.Errorf("we already have a connection")
	}

	if packet.sequence > c.expectedRecvSeq {
		return nil, fmt.Errorf("packet received out of order")
	}

	// We now know the client ID
	c.destId = packet.sourceMachine
	c.state = RESPONDED
	c.peer = source

	// Send response
	response := c.NewConnectionResponse()

	// Time how long we take to get a response
	c.ackSent = time.Now()

	return &response, nil
}

func (c *Connection) handleResponse(packet ConnectionPacket) (*ConnectionPacket, error) {
	// Ignore if it doesn't match our request
	if c.state != REQUESTED {
		return nil, fmt.Errorf("we did not request a connection")
	}

	// We now know the client ID
	c.destId = packet.sourceMachine
	c.state = ESTABLISHED

	// Ack it
	response := c.NewConnectionAck(packet.sequence)

	// This is treated as an ack
	c.goBackNAck()

	// We got a response, we need to wait at least this long before considering a packet lost
	c.ackRecv = time.Now()

	return &response, nil
}

func (c *Connection) handleAck(packet ConnectionPacket) (bool, error) {
	connectionStatusChanged := false

	// If this does not match our state, we ignore
	if c.state == IDLE || c.state == REQUESTED {
		return false, fmt.Errorf("we do not have an active connection")
	}

	if len(c.sendWindow) < 1 || packet.sequence != c.sendWindow[0].sequence {
		logging.Debugf("ack rejected out of order acked %d", packet.sequence)
		if len(c.sendWindow) < 1 {
			logging.Debugf("we have no sent\n")
		} else {
			logging.Debugf(" expected %d\n", c.sendWindow[0].sequence)
		}
		return false, fmt.Errorf("received out of order ack")
	}

	if c.state == RESPONDED {
		// We got a response, we need to wait at least this long before considering a packet lost
		c.ackRecv = time.Now()
		c.state = ESTABLISHED
		// Since the connection is now established, this is a new connection
		connectionStatusChanged = true
	}

	c.goBackNAck()

	return connectionStatusChanged, nil
}

func (c *Connection) handleData(packet ConnectionPacket) (*ConnectionPacket, []byte, error) {
	if c.state == IDLE || c.state == REQUESTED {
		return nil, nil, fmt.Errorf("we do not have an active connection")
	}

	// Send an ack
	response := c.NewConnectionAck(packet.sequence)

	// Return data if this is a new packet
	var extraData []byte = nil
	if packet.sequence == c.expectedRecvSeq {
		extraData = packet.data
	}
	logging.Debugf("received data")

	return &response, extraData, nil
}

func (c *Connection) goBackNAck() {
	// Update go back n
	c.sendWindow = c.sendWindow[1:]
	c.windowPos--
	c.numLosses = 0
	if len(c.sendWindow) > 0 {
		// Reset the deadline grace period since we have more packets
		c.setDeadline()
	} else {
		// We have nothing to send, so we have no deadline
		c.lossDeadline = time.Time{}
	}
}

func (c ConnectionPacket) serialize() ([]byte, error) {
	logging.Debugf("serializing connection packet of type %d sequence %d\n", c.packetType, c.sequence)
	buf := bytes.Buffer{}

	// Write the source machine ID, 16 bytes
	err := binary.Write(&buf, binary.BigEndian, c.sourceMachine)
	if err != nil {
		return nil, err
	}

	// Write the destination machine ID, 16 bytes
	err = binary.Write(&buf, binary.BigEndian, c.destMachine)
	if err != nil {
		return nil, err
	}

	// Write the sequence number, 4 bytes
	err = binary.Write(&buf, binary.BigEndian, c.sequence)
	if err != nil {
		return nil, err
	}

	// Write the packet type, 4 bytes
	err = binary.Write(&buf, binary.BigEndian, c.packetType)
	if err != nil {
		return nil, err
	}

	// Write the data as is
	buf.Write(c.data)

	return buf.Bytes(), nil
}

func connectionDeserialize(buf []byte) (ConnectionPacket, error) {
	connectionPacket := ConnectionPacket{}
	headerSize := binary.Size(ConnectionHeader{})

	// Ensure we have enough data to read
	if len(buf) < headerSize {
		return ConnectionPacket{}, fmt.Errorf("error reading connection packet header: not long enough")
	}

	reader := bytes.NewReader(buf[:headerSize])

	// Read the source machine ID, 16 bytes
	err := binary.Read(reader, binary.BigEndian, &connectionPacket.sourceMachine)
	if err != nil {
		return ConnectionPacket{}, err
	}

	// Read the destination machine ID, 16 bytes
	err = binary.Read(reader, binary.BigEndian, &connectionPacket.destMachine)
	if err != nil {
		return ConnectionPacket{}, err
	}

	// Read the sequence number, 4 bytes
	err = binary.Read(reader, binary.BigEndian, &connectionPacket.sequence)
	if err != nil {
		return ConnectionPacket{}, err
	}

	// Read the packet type, 4 bytes
	err = binary.Read(reader, binary.BigEndian, &connectionPacket.packetType)
	if err != nil {
		return ConnectionPacket{}, err
	}

	// Read the rest of the data as is
	connectionPacket.data = buf[headerSize:]

	return connectionPacket, nil
}
