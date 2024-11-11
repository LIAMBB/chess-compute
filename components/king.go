package components

type King struct {
	Color bool // white = true, black = false
}

func (king King) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	var possibleBoards []*ChessBoard

	// Define all possible moves for a king
	moves := []Coordinates{
		{X: position.X + 1, Y: position.Y},
		{X: position.X - 1, Y: position.Y},
		{X: position.X, Y: position.Y + 1},
		{X: position.X, Y: position.Y - 1},
		{X: position.X + 1, Y: position.Y + 1},
		{X: position.X + 1, Y: position.Y - 1},
		{X: position.X - 1, Y: position.Y + 1},
		{X: position.X - 1, Y: position.Y - 1},
	}

	// var cachedPossibleAttacks []*ChessBoard
	for _, move := range moves {
		if board.IsWithinBounds(move) && (board.IsEmpty(move) || board.IsEnemy(move, king.Color)) {
			newBoard := board.DeepCopy()
			newBoard.MovePiece(position, move)
			// Ensure the move doesn't leave the king in check
			if !newBoard.WouldLeaveKingInCheck(king.Color, nil) {
				possibleBoards = append(possibleBoards, newBoard)
			}
		}
	}

	return possibleBoards
}

func (king King) GetColor() bool {
	return king.Color
}

func (king King) ToString() string {
	if king.Color {
		return " W K "
	} else {
		return " B K "
	}
}
