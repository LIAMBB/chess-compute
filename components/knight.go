package components

// type ChessPiece interface {
// 	GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []Coordinates
// 	GetColor() bool
// }

type Knight struct {
	Color bool // white = true, black = false
}

func (knight Knight) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []ChessBoard {
	// Vertical (y +/- 2) (x +/- 1)
	// Horizontal (y +/- 1) (x +/- 2)
	return []ChessBoard{}
}

func (knight Knight) GetColor() bool {
	return knight.Color
}

func (knight Knight) IsEmpty() bool {
	return false
}

func (knight Knight) ToString() string {
	if knight.Color {
		return " W N " // Knight is N to prevent confusing with King
	} else {
		return " B N "
	}
}
