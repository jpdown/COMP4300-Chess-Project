package chess

import (
	"project-go/logging"
)

type Board struct {
	State [8][8]IPiece
}

func Generate() Board {
	board := Board{}
	board.State[0][0] = NewRook(BLACK)
	board.State[0][1] = NewKnight(BLACK)
	board.State[0][2] = NewBishop(BLACK)
	board.State[0][3] = NewQueen(BLACK)
	board.State[0][4] = NewKing(BLACK)
	board.State[0][5] = NewBishop(BLACK)
	board.State[0][6] = NewKnight(BLACK)
	board.State[0][7] = NewRook(BLACK)

	// The second row for each board is filled with pawns
	for i := 0; i < 8; i++ {
		board.State[1][i] = NewPawn(BLACK)
	}
	for i := 0; i < 8; i++ {
		board.State[6][i] = NewPawn(WHITE)
	}

	board.State[7][0] = NewRook(WHITE)
	board.State[7][1] = NewKnight(WHITE)
	board.State[7][2] = NewBishop(WHITE)
	board.State[7][3] = NewQueen(WHITE)
	board.State[7][4] = NewKing(WHITE)
	board.State[7][5] = NewBishop(WHITE)
	board.State[7][6] = NewKnight(WHITE)
	board.State[7][7] = NewRook(WHITE)

	return board
}

func (b Board) Print() {
	// Print the column letters
	logging.Log("   a  b  c  d  e  f  g  h\n")

	for row := 0; row < len(b.State); row++ {
		// Print the row number
		logging.Logf("%d ", row)

		for col := 0; col < len(b.State[row]); col++ {
			piece := b.State[row][col]

			if piece != nil {
				logging.Logf(b.State[row][col].Representation() + " ")
			} else {
				// Empty space is represented by a dot
				logging.Logf(" . ")
			}
		}
		// Print a new line since this row is finished
		logging.Log("")
	}
}
