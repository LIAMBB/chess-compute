package components

// type ChessPiece interface {
// 	GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []Coordinates
// 	GetColor() bool
// }

type Pawn struct {
	Color bool // white = true, black = false
}

func (pawn Pawn) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []ChessBoard {
	// Forward 1
	// (First Move) Forward 2
	// (Capture) Diagonal 1 (L/R)
	return []ChessBoard{}
}

func (pawn Pawn) GetColor() bool {
	return pawn.Color
}

func (pawn Pawn) IsEmpty() bool {
	return false
}

func (pawn Pawn) ToString() string {
	if pawn.Color {
		return " W P "
	} else {
		return " B P "
	}
}
