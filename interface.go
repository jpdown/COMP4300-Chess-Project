package main

import (
	"fmt"
	"project-go/chess"
	"project-go/logging"
	"project-go/networking"
	"strings"
)

func PrintPrompt(ctx *Context) {
	// Call the correct prompt function based on the current state
	switch ctx.ClientState {
	case MENU:
		mainMenuPrompt()
		break
	case LOBBY:
		lobbyPrompt(ctx)
		break
	case MY_TURN:
		myTurnPrompt(ctx)
		break
	case THEIR_TURN:
		theirTurnPrompt(ctx)
		break
	}
}

func HandleInput(ctx *Context, input string) ClientState {
	// Remove the newline from the end of the input
	input = strings.TrimSuffix(input, "\n")

	// Parse the input correctly based on the current state
	switch ctx.ClientState {
	case MENU:
		return mainMenuInput(ctx, input)
	case LOBBY:
		return lobbyInput(ctx, input)
	case MY_TURN:
		return myTurnInput(ctx, input)
	case THEIR_TURN:
		return theirTurnInput(ctx, input)
	default:
		logging.Log("WE ARE IN AN INVALID STATE")
		return ctx.ClientState
	}
}

func mainMenuPrompt() {
	logging.Log(".start <name> - Starts a new game")
	logging.Log(".list - Lists existing games")
	logging.Log(".join <name> - Joins existing games")
}

func mainMenuInput(ctx *Context, input string) ClientState {
	// Split on space to parse the extra arguments if necessary
	split := strings.Split(input, " ")

	switch split[0] {
	case ".start":
		if len(split) < 2 {
			logging.Log("Please enter a name for the lobby. eg. .start thegame")
			return MENU
		}

		// Create the lobby
		ctx.Lobby = CreateLobby(split[1])
		packet := networking.NewLobbyCreated(split[1])

		// Broadcast that the lobby exists
		err := ctx.BroadcastPacket(packet)
		if err != nil {
			logging.Log("Error creating the lobby.")
			return MENU
		}

		return LOBBY
	case ".list":
		logging.Log("Asking once for lobbies...")
		packet := networking.NewLobbyListRequest()

		// Ask everyone for lobbies
		err := ctx.BroadcastPacket(packet)
		if err != nil {
			logging.Log("Error asking for lobbies.")
		}

		return MENU
	case ".join":
		if len(split) < 2 {
			logging.Log("Please enter a name for the lobby. eg. .join thegame")
		}
		logging.Log("Attempting to join...")
		packet := networking.NewLobbyJoinRequest(split[1])

		// Broadcast that we want to join the lobby with the given name
		err := ctx.BroadcastPacket(packet)
		if err != nil {
			logging.Log("Error joining.")
		}

		return MENU
	default:
		logging.Log("Invalid command.")
		return MENU
	}
}

func lobbyPrompt(ctx *Context) {
	if ctx.Lobby.hosting {
		logging.Log(".start - Starts the game (requires other player)")
	}
	logging.Log(".leave - Leaves the game")
}

func lobbyInput(ctx *Context, input string) ClientState {
	// Split on space to parse the extra arguments if necessary
	split := strings.Split(input, " ")

	switch split[0] {
	case ".start":
		if !ctx.Lobby.hosting {
			logging.Log("You are not the host!")
			return LOBBY
		}

		if !ctx.Connection.IsActive() {
			logging.Log("There isn't a second player!")
			return LOBBY
		}

		logging.Log("Attempting to start game...")
		ctx.Lobby.Ready = true
		packet := networking.NewLobbyStartRequest()

		// Tell our peer that we want to start
		err := ctx.SendPacket(packet)
		if err != nil {
			logging.Log("Error starting the game.")
		}

		return LOBBY
	case ".leave":
		if ctx.Connection.IsActive() {
			logging.Log("connection is active, terminating it")
			ctx.Connection.Close()
		}

		// We no longer have a lobby, fully clear our state
		ctx.Lobby = Lobby{}
		return MENU
	default:
		logging.Log("Invalid command.")
		return LOBBY
	}
}

func myTurnPrompt(ctx *Context) {
	ctx.GameState.Print()
	logging.Log("")
	logging.Logf("IT IS YOUR TURN, YOU ARE ")
	ctx.GameState.PrintTurn()
	logging.Log(".move <src> <dest> - Moves a piece, eg .move A4 B3")
	logging.Log(".forfeit - Forfeits the game")
}

func myTurnInput(ctx *Context, input string) ClientState {
	// Split on space to parse the extra arguments if necessary
	split := strings.Split(input, " ")

	switch split[0] {
	case ".move":
		if len(split) < 3 {
			logging.Log("Please enter a move in the correct format")
			return MY_TURN
		}

		// Parse the desired move from the input
		srcPos, destPos, err := parseMovement(split[1], split[2])
		if err != nil {
			logging.Log("Please enter a move in the correct format")
			return MY_TURN
		}

		// Try to move the piece
		moved, failedReason := ctx.GameState.MovePiece(srcPos, destPos)
		if !moved {
			logging.Log(failedReason)
			return MY_TURN
		}

		// Moving the piece was successful, so it is no longer our turn
		ctx.GameState.SwitchTurn()

		// Tell our peer what movement was made
		packet := networking.NewMovePiece(srcPos, destPos)
		err = ctx.SendPacket(packet)
		if err != nil {
			logging.Log("Error moving the piece.")
			return MY_TURN
		}
		return THEIR_TURN
	case ".forfeit":
		// Tell our peer that we forfeit
		packet := networking.NewForfeit()
		err := ctx.SendPacket(packet)
		if err != nil {
			logging.Log("Error forfeiting.")
		}
		logging.Log("You have forfeit the match.")
		return MENU
	default:
		logging.Log("Invalid command.")
		return MY_TURN
	}
}

func theirTurnPrompt(ctx *Context) {
	ctx.GameState.Print()
	logging.Log("")
	logging.Logf("IT IS THEIR TURN, THEY ARE ")
	ctx.GameState.PrintTurn()
	logging.Log(".forfeit - Forfeits the game")
}

func theirTurnInput(ctx *Context, input string) ClientState {
	// Split on space to parse the extra arguments if necessary
	split := strings.Split(input, " ")

	switch split[0] {
	case ".forfeit":
		// Tell our peer that we forfeit
		packet := networking.NewForfeit()
		err := ctx.SendPacket(packet)
		if err != nil {
			logging.Log("Error forfeiting.")
		}
		logging.Log("You have forfeit the match.")
		return MENU
	default:
		logging.Log("Invalid command.")
		return THEIR_TURN
	}
}

func parseMovement(src string, dest string) (chess.Position, chess.Position, error) {
	// We are looking for input in the form letternumber
	// E.g. a4 b3
	// Letters in range a-h
	// Numbers in range 1-8

	var srcRow, destRow int
	// This is a rune to allow easy letter parsing
	var srcColRune, destColRune rune

	// Pull the column and row out of the src string
	_, err := fmt.Sscanf(src, "%c%1d", &srcColRune, &srcRow)
	if err != nil {
		return chess.Position{}, chess.Position{}, err
	}

	// Pull the column and row out of the dest string
	_, err = fmt.Sscanf(dest, "%c%1d", &destColRune, &destRow)
	if err != nil {
		return chess.Position{}, chess.Position{}, err
	}

	// Parse the columns from the letters
	srcCol := parseLetter(srcColRune)
	destCol := parseLetter(destColRune)

	// Ensure both positions are in bounds
	if !inRange(srcCol, 0, 8) || !inRange(srcRow, 0, 8) || !inRange(destCol, 0, 8) || !inRange(destRow, 0, 8) {
		return chess.Position{}, chess.Position{}, fmt.Errorf("incorrect piece position")
	}

	return chess.Position{X: srcCol, Y: srcRow}, chess.Position{X: destCol, Y: destRow}, nil
}

func parseLetter(letter rune) int {
	// 'a' is 0, so subtract it from the given letter
	return int(letter - 'a')
}

func inRange(pos int, min int, max int) bool {
	return pos >= min && pos < max
}
