package chess

import "project-go/util"

// Source https://en.wikipedia.org/wiki/Rules_of_chess

type Position struct {
	X int
	Y int
}

type Movement struct {
	old       Position
	new       Position
	wouldTake bool
}

type Colour int

const (
	WHITE Colour = iota
	BLACK
)

type Piece struct {
	MovementCollision bool
	HasMoved          bool
	text              string
	colour            Colour
}

type IPiece interface {
	CanMove(movement Movement) bool
	Representation() string
	Colour() Colour
	HasMovementCollision() bool
	Moved()
}

type King struct {
	Piece
}
type Queen struct {
	Piece
}
type Rook struct {
	Piece
}
type Bishop struct {
	Piece
}
type Knight struct {
	Piece
}
type Pawn struct {
	Piece
}

func NewKing(colour Colour) *King {
	return &King{Piece{MovementCollision: true, HasMoved: false, text: "K", colour: colour}}
}

func NewQueen(colour Colour) *Queen {
	return &Queen{Piece{MovementCollision: true, HasMoved: false, text: "Q", colour: colour}}
}

func NewRook(colour Colour) *Rook {
	return &Rook{Piece{MovementCollision: true, HasMoved: false, text: "R", colour: colour}}
}

func NewBishop(colour Colour) *Bishop {
	return &Bishop{Piece{MovementCollision: true, HasMoved: false, text: "B", colour: colour}}
}

func NewKnight(colour Colour) *Knight {
	return &Knight{Piece{MovementCollision: false, HasMoved: false, text: "H", colour: colour}}
}

func NewPawn(colour Colour) *Pawn {
	return &Pawn{Piece{MovementCollision: true, HasMoved: false, text: "P", colour: colour}}
}

func (k King) CanMove(movement Movement) bool {
	// Kings can only move one square
	return moveDistance(movement) <= 1
}

func (q Queen) CanMove(movement Movement) bool {
	// Queens can move in any straight line, any distance
	return movingDiagonal(movement) || movingCardinal(movement)
}

func (r Rook) CanMove(movement Movement) bool {
	// Rooks can move any distance, but no diagonals
	return movingCardinal(movement)
}

func (b Bishop) CanMove(movement Movement) bool {
	// Bishops can move any distance, but no cardinals
	return movingDiagonal(movement)
}

func (k Knight) CanMove(movement Movement) bool {
	// L shapes
	// one direction 2, other direction 1

	xDiff := util.Abs(movement.new.X - movement.old.X)
	yDiff := util.Abs(movement.new.Y - movement.old.Y)

	return (xDiff == 2 && yDiff == 1) || (xDiff == 1 && yDiff == 2)
}

func (p Pawn) CanMove(movement Movement) bool {
	// This needs a special case for if it would be taking
	// if taking, can move diagonal
	// if not taking, cannot move diagonal

	yDiff := movement.new.Y - movement.old.Y

	// Pawns that have moved can only move one space
	distance := moveDistance(movement)
	if p.HasMoved && distance > 1 {
		return false
	}

	// If the pawn has not moved yet, it is allowed to move 2 spaces
	if !p.HasMoved && distance > 2 {
		return false
	}

	// If the pawn is moving 2 spaces, it cannot move diagonal
	if distance == 2 && movingDiagonal(movement) {
		return false
	}

	// Moving diagonal when not taking is invalid
	if !movement.wouldTake && movingDiagonal(movement) {
		return false
	}

	// Not moving diagonal when taking is invalid
	if movement.wouldTake && !movingDiagonal(movement) {
		return false
	}

	// can only move forward - up if white, down if black
	return (p.colour == BLACK && yDiff > 0) || (p.colour == WHITE && yDiff < 0)
}

func (p *Piece) Representation() string {
	var colourText string
	if p.Colour() == WHITE {
		colourText = "w"
	} else {
		colourText = "b"
	}
	return colourText + p.text
}

func (p *Piece) Colour() Colour {
	return p.colour
}

func (p *Piece) HasMovementCollision() bool {
	return p.MovementCollision
}

func (p *Piece) Moved() {
	p.HasMoved = true
}

func movingDiagonal(movement Movement) bool {
	xDiff := movement.new.X - movement.old.X
	yDiff := movement.new.Y - movement.old.Y

	// Movement is diagonal if X and Y distance are identical
	return util.Abs(xDiff) == util.Abs(yDiff)
}

func movingCardinal(movement Movement) bool {
	xDiff := movement.new.X - movement.old.X
	yDiff := movement.new.Y - movement.old.Y

	// Movement is cardinal if one of the directions is zero
	return xDiff == 0 || yDiff == 0
}

func moveDistance(movement Movement) int {
	xDiff := movement.new.X - movement.old.X
	yDiff := movement.new.Y - movement.old.Y

	// Choose the maximum, this is only useful for straight line moves
	// That is, any piece that is not the knight
	return util.Max(util.Abs(xDiff), util.Abs(yDiff))
}
