package components

type ChessPiece interface {
	GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard
	GetColor() bool
	ToString() string
}
