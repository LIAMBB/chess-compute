package components

type ChessPiece interface {
	GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []ChessBoard
	GetColor() bool
	IsEmpty() bool
	ToString() string
}

type EmptySpace struct {
}

func (empty EmptySpace) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []ChessBoard {
	return []ChessBoard{}
}

func (empty EmptySpace) GetColor() bool {
	return false
}

func (empty EmptySpace) IsEmpty() bool {
	return true
}

func (empty EmptySpace) ToString() string {
	return " O O "
}
