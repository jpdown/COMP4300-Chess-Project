package chess

import "project-go/logging"

type State struct {
	board Board
	turn  Colour
}

func CreateState() State {
	state := State{board: Generate(), turn: WHITE}

	return state
}

func (s *State) Print() {
	s.board.Print()
}

func (s *State) MovePiece(source Position, dest Position) (bool, string) {
	piece := s.board.State[source.Y][source.X]

	if piece == nil {
		return false, "There is no piece there"
	}

	if piece.Colour() != s.turn {
		return false, "That is not your piece"
	}

	collidingPiece := s.board.State[dest.Y][dest.X]
	// Colliding with an enemy piece is fine
	if collidingPiece != nil && collidingPiece.Colour() == piece.Colour() {
		return false, "Would collide with your own piece"
	}

	movement := Movement{old: source, new: dest}
	if piece.HasMovementCollision() && s.wouldCollide(movement) {
		return false, "There is a piece in that path"
	}

	movement.wouldTake = collidingPiece != nil && collidingPiece.Colour() != piece.Colour()
	if !piece.CanMove(movement) {
		return false, "That piece cannot move there"
	}

	// CHECK DETECTION
	// First, apply the movement. We will revert this if it results in check
	s.board.State[dest.Y][dest.X] = piece
	s.board.State[source.Y][source.X] = nil

	if s.kingInCheck() {
		// The King is in check, revert this movement
		s.board.State[source.Y][source.X] = piece
		s.board.State[dest.Y][dest.X] = nil
		return false, "That move results in check"
	}

	// Tell the piece that it has moved
	// This is used so Pawns can move forward 2 only if they have not moved
	piece.Moved()

	return true, ""
}

func (s *State) SwitchTurn() {
	if s.turn == WHITE {
		s.turn = BLACK
	} else {
		s.turn = WHITE
	}
}

func (s *State) PrintTurn() {
	if s.turn == WHITE {
		logging.Log("WHITE")
	} else {
		logging.Log("BLACK")
	}
}

func (s *State) kingInCheck() bool {
	// If any of the enemy pieces can move onto the King, this is invalid
	// First, find the King
	kingPos, ok := s.findKing()

	// King was not found, it is not in check
	if !ok {
		return false
	}

	// Now, loop over every piece on the board
	// If they are an enemy piece, see if they can move to the King
	for row := 0; row < len(s.board.State); row++ {
		for col := 0; col < len(s.board.State[row]); col++ {
			enemyPiece := s.board.State[row][col]

			// Ignore empty spaces and our own pieces
			if enemyPiece == nil || enemyPiece.Colour() == s.turn {
				continue
			}

			// We found an enemy piece, see if they can move to the King
			enemyMove := Movement{
				old:       Position{X: col, Y: row},
				new:       kingPos,
				wouldTake: true,
			}

			canMove := enemyPiece.CanMove(enemyMove)
			hasCollision := enemyPiece.HasMovementCollision()

			if canMove && (!hasCollision || !s.wouldCollide(enemyMove)) {
				// The King is in check
				return true
			}
		}
	}

	// The King is not in check
	return false
}

func (s *State) findKing() (Position, bool) {
	for row := 0; row < len(s.board.State); row++ {
		for col := 0; col < len(s.board.State[row]); col++ {
			piece := s.board.State[row][col]

			// Try to cast it to a King
			_, pieceIsKing := piece.(*King)

			// If this piece is our King, return the pos
			if piece != nil && pieceIsKing && piece.Colour() == s.turn {
				return Position{X: col, Y: row}, true
			}
		}
	}

	// Return an empty position if unable to be found
	return Position{}, false
}

func (s *State) wouldCollide(movement Movement) bool {
	xDiff := movement.new.X - movement.old.X
	yDiff := movement.new.Y - movement.old.Y

	xDirection := multipliableDirection(xDiff)
	yDirection := multipliableDirection(yDiff)

	// Move the piece in the movement direction, checking each spot for the existence of a piece
	for offset := 1; movement.old.X+(offset*xDirection) != movement.new.X || movement.old.Y+(offset*yDirection) != movement.new.Y; offset++ {
		xPos := movement.old.X + (offset * xDirection)
		yPos := movement.old.Y + (offset * yDirection)
		if s.board.State[yPos][xPos] != nil {
			return true
		}
	}

	return false
}

func multipliableDirection(diff int) int {
	if diff > 0 {
		return 1
	} else if diff < 0 {
		return -1
	} else {
		return 0
	}
}
