package components

// type ChessPiece interface {
// 	GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []Coordinates
// 	GetColor() bool
// }

type Bishop struct {
	Color bool // white = true, black = false
}

func (bishop Bishop) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	// +x linear line

	// -x linear line
	return []*ChessBoard{}
}

func (bishop Bishop) GetColor() bool {
	return bishop.Color
}

func (bishop Bishop) ToString() string {
	if bishop.Color {
		return " W B "
	} else {
		return " B B "
	}
}
