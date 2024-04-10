package main

import (
	"project-go/chess"
	"project-go/logging"
	"project-go/networking"
)

func HandlePacket(ctx *Context, packet networking.IChessPacket) ClientState {
	// Call the correct handler based on the packet type
	switch casted := packet.(type) {
	case networking.LobbyCreatedPacket:
		return handleLobbyCreated(ctx, casted)
	case networking.LobbyListRequest:
		return handleLobbyListRequest(ctx, casted)
	case networking.LobbyInfoPacket:
		return handleLobbyInfo(ctx, casted)
	case networking.LobbyJoinRequest:
		return handleLobbyJoinRequest(ctx, casted)
	case networking.LobbyStartRequest:
		return handleLobbyStartRequest(ctx, casted)
	case networking.LobbyStartAccepted:
		return handleLobbyStartAccepted(ctx, casted)
	case networking.MovePiecePacket:
		return handleMovePiece(ctx, casted)
	case networking.ForfeitPacket:
		return handleForfeit(ctx, casted)
	default:
		return ctx.ClientState
	}
}

func handleLobbyCreated(ctx *Context, packet networking.LobbyCreatedPacket) ClientState {
	if ctx.ClientState == MENU {
		logging.Log("New lobby created: " + packet.Name)
	}
	return ctx.ClientState
}

func handleLobbyListRequest(ctx *Context, packet networking.LobbyListRequest) ClientState {
	var response networking.LobbyInfoPacket
	if ctx.Lobby.hosting && !ctx.Connection.IsActive() {
		// Respond to the broadcast with another broadcast announcing our lobby
		response = networking.NewLobbyInfo(ctx.Lobby.name)
		err := ctx.BroadcastPacket(response)
		if err != nil {
			logging.Log("Error broadcasting lobby info.")
		}
	}
	return ctx.ClientState
}

func handleLobbyInfo(ctx *Context, packet networking.LobbyInfoPacket) ClientState {
	if ctx.ClientState == MENU {
		logging.Log("Lobby available at: " + packet.Name)
	}
	return ctx.ClientState
}

func handleLobbyJoinRequest(ctx *Context, packet networking.LobbyJoinRequest) ClientState {
	if ctx.Lobby.hosting && !ctx.Connection.IsActive() && packet.Name == ctx.Lobby.name {
		logging.Logf("Peer %x is trying to join your game.\n", packet.SourceAddress)

		// Try to open a connection with the peer that wants to join
		// Upon a successful connection, the game will be ready to start
		err := ctx.Connection.Open(packet.SourceAddress)
		if err != nil {
			logging.Debug("Error opening connection: " + err.Error())
			return ctx.ClientState
		}

		return LOBBY
	}

	return ctx.ClientState
}

func handleLobbyStartRequest(ctx *Context, packet networking.LobbyStartRequest) ClientState {
	if !ctx.Lobby.hosting {
		// Tell the lobby host that we're ok to start the game
		response := networking.NewLobbyStartAccepted()
		err := ctx.SendPacket(response)
		if err != nil {
			logging.Debug("error sending start lobby: " + err.Error())
		}

		// Reset the game state before showing the game board
		ctx.GameState = chess.CreateState()
		return THEIR_TURN
	}

	return ctx.ClientState
}

func handleLobbyStartAccepted(ctx *Context, packet networking.LobbyStartAccepted) ClientState {
	if ctx.Lobby.hosting && ctx.Lobby.Ready {
		logging.Log("Game is starting")
		// Reset the game state before showing the game board
		ctx.GameState = chess.CreateState()
		return MY_TURN
	}
	return ctx.ClientState
}

func handleMovePiece(ctx *Context, packet networking.MovePiecePacket) ClientState {
	// Move the piece switch turns, we're ready to accept user input again
	ctx.GameState.MovePiece(packet.SrcPos, packet.DestPos)
	ctx.GameState.SwitchTurn()
	return MY_TURN
}

func handleForfeit(ctx *Context, packet networking.ForfeitPacket) ClientState {
	logging.Log("The other user has forfeit.")
	ctx.Lobby.hosting = false
	ctx.Connection.Close()
	return MENU
}
