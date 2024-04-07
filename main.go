package main

import (
	"project-go/chess"
	"project-go/logging"
	"project-go/networking"
	"time"
)

type ClientState int

const (
	MENU ClientState = iota
	LOBBY
	MY_TURN
	THEIR_TURN
	EXITING
)

type Context struct {
	GameState   chess.State
	ClientState ClientState
	Lobby       Lobby
	Connection  *networking.Connection
}

func main() {
	// Set up our context that lives through the entire runtime
	context := Context{
		GameState:   chess.CreateState(),
		ClientState: MENU,
		Connection:  networking.NewConnection(),
	}

	// Initialize the logger
	logging.Init()

	// Set up our channels for the threads we're running
	networking.SendChan = make(chan []byte)
	go networking.SendThread()

	inputChan := make(chan string)
	go InputThread(inputChan)

	receiveChan := make(chan []byte)
	go networking.RecvThread(receiveChan)

	tickChan := make(chan byte)
	go tickThread(&context, tickChan)

	// Output the prompt for the user
	PrintPrompt(&context)

	for context.ClientState != EXITING {
		select {
		case input := <-inputChan:
			context.handleInput(input)
			break
		case frame := <-receiveChan:
			connectionStatusChanged, packet, err := networking.HandleFrame(frame, context.Connection)

			// If we have a valid error that isn't just that we received a frame with the wrong ethertype, log it
			if err != nil && err.Error() != "incorrect ethertype" {
				logging.Debug("Error receiving packet, " + err.Error())
				continue
			}

			if packet != nil {
				logging.Debugf("received packet: %x\n", packet)
				context.handleRequest(packet)
			}

			if connectionStatusChanged {
				context.handleConnectionChange()
			}
			break
		case _ = <-tickChan:
			// We don't do anything if we don't have an active connection
			if !context.Connection.IsActive() {
				continue
			}

			// Update our connection as necessary for this tick
			context.tickConnection()
			break
		}
	}
}

func (c *Context) handleInput(input string) {
	newState := HandleInput(c, input)

	if newState != c.ClientState {
		c.changeState(newState)
	}
}

func (c *Context) handleRequest(request networking.IChessPacket) {
	newState := HandlePacket(c, request)

	if newState != c.ClientState {
		c.changeState(newState)
	}
}

func (c *Context) handleConnectionChange() {
	if c.Connection.IsActive() {
		logging.Logf("Got a new connection with %x, entering the lobby\n", c.Connection.Peer())
		// We just received a new connection, meaning we have joined the lobby
		c.changeState(LOBBY)
		return
	} else {
		logging.Log("Other side closed the connection.")
		// If we were in the middle of a game, reset back to the menu
		if c.ClientState == MY_TURN || c.ClientState == THEIR_TURN || !c.Lobby.hosting {
			c.changeState(MENU)
			// The lobby has closed, so purge all state
			c.Lobby = Lobby{}
		}
	}

	PrintPrompt(c)
}

func (c *Context) tickConnection() {
	// Send packets if needed
	closed := c.Connection.CheckLoss()
	if closed {
		logging.Log("Connection timed out.")
		c.changeState(MENU)
		return
	}

	// Get a full list of packets to send on the wire
	packets := c.Connection.GetPackets()
	packets = append(packets, c.Connection.GetAckPackets()...)

	for i := range packets {
		// Package it up for sending on the wire
		data, err := networking.PackageTransport(packets[i], c.Connection)
		if err != nil {
			logging.Debug("error packaging transport packet: " + err.Error())
			continue
		}

		// There was no error, send the packet on the wire
		networking.SendChan <- data
	}
}

func (c *Context) changeState(state ClientState) {
	c.ClientState = state
	PrintPrompt(c)
}

func (c *Context) SendPacket(packet networking.IChessPacket) error {
	connData, err := networking.PackageChess(packet, c.Connection)
	if err != nil {
		return err
	}

	// Queue the packet to send as soon as the connection is ready (hopefully next tick)
	c.Connection.QueuePacket(connData)
	return nil
}

func (c *Context) BroadcastPacket(packet networking.IChessPacket) error {
	data, err := networking.PackageChessBroadcast(packet)
	if err != nil {
		return err
	}

	networking.SendChan <- data
	return nil
}

func tickThread(context *Context, tickChan chan byte) {
	for context.ClientState != EXITING {
		// Tick to wake up the main thread 20 times a second
		time.Sleep(time.Millisecond * 50)
		tickChan <- 1
	}
}
