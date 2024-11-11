package components

// type ChessPiece interface {
// 	GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []Coordinates
// 	GetColor() bool
// }

type Rook struct {
	Color bool // white = true, black = false
}

func (rook Rook) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	// Vertical
	// Horizontal
	return []*ChessBoard{}
}

func (rook Rook) GetColor() bool {
	return rook.Color
}

func (rook Rook) IsEmpty() bool {
	return false
}

func (rook Rook) ToString() string {
	if rook.Color {
		return " W R "
	} else {
		return " B R "
	}
}
