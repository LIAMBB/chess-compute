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

	// Create a cache for opponent attacks
	cache := &AttackCache{
		Positions: make(map[Coordinates]bool),
		Computed:  false,
	}

	for _, move := range moves {
		if board.IsWithinBounds(move) && (board.IsEmpty(move) || board.IsEnemy(move, king.Color)) {
			newBoard := board.DeepCopy()
			newBoard.MovePiece(position, move)

			// Only check if the move is safe if we're not already checking for check
			if !inCheck {
				if !newBoard.WouldLeaveKingInCheck(king.Color, cache, 0) {
					possibleBoards = append(possibleBoards, newBoard)
				}
			} else {
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
