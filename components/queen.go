package components

// type ChessPiece interface {
// 	GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []Coordinates
// 	GetColor() bool
// }

type Queen struct {
	Color bool // white = true, black = false
}

func (queen Queen) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	// Vertical
	// Horizontal
	// +x linear
	// -x linear
	return []*ChessBoard{}
}

func (queen Queen) GetColor() bool {
	return queen.Color
}

func (queen Queen) ToString() string {
	if queen.Color {
		return " W Q "
	} else {
		return " B Q "
	}
}
