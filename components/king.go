package components

// type ChessPiece interface {
// 	GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []Coordinates
// 	GetColor() bool
// }

type King struct {
	Color bool // white = true, black = false
}

func (king King) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	// N
	// nE
	// E
	// sE
	// S
	// sW
	// W
	//nW
	return []*ChessBoard{}
}

func (king King) GetColor() bool {
	return king.Color
}
