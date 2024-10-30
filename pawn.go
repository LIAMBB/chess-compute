package main

// type ChessPiece interface {
// 	GetPossibleMoves(board ChessBoard, position Coordinates) []Coordinates
// 	GetColor() bool
// }

type Pawn struct {
	Color bool // white = true, black = false
}

func (pawn Pawn) GetPossibleMoves(board ChessBoard, position Coordinates) []Coordinates {
	// Forward 1
	// (First Move) Forward 2
	// (Capture) Diagonal 1 (L/R)
	return []Coordinates{}
}

func (pawn Pawn) GetColor() bool {
	return pawn.Color
}

func (pawn Pawn) isEmpty() bool {
	return false
}
