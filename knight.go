package main

// type ChessPiece interface {
// 	GetPossibleMoves(board ChessBoard, position Coordinates) []Coordinates
// 	GetColor() bool
// }

type Knight struct {
	Color bool // white = true, black = false
}

func (knight Knight) GetPossibleMoves(board ChessBoard, position Coordinates) []Coordinates {
	// Vertical (y +/- 2) (x +/- 1)
	// Horizontal (y +/- 1) (x +/- 2)
	return []Coordinates{}
}

func (knight Knight) GetColor() bool {
	return knight.Color
}

func (knight Knight) IsEmpty() bool {
	return false
}
