package components

type Rook struct {
	Color bool // white = true, black = false
}

func (rook Rook) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	var possibleBoards []*ChessBoard

	// Define directions for rook movement
	directions := []Coordinates{
		{X: 1, Y: 0},  // Right
		{X: -1, Y: 0}, // Left
		{X: 0, Y: 1},  // Down
		{X: 0, Y: -1}, // Up
	}

	for _, direction := range directions {
		for i := 1; i < 8; i++ {
			newPosition := Coordinates{X: position.X + i*direction.X, Y: position.Y + i*direction.Y}
			if !board.IsWithinBounds(newPosition) {
				break
			}
			if board.IsEmpty(newPosition) {
				newBoard := board.DeepCopy()
				newBoard.MovePiece(position, newPosition)
				possibleBoards = append(possibleBoards, newBoard)
			} else if board.IsEnemy(newPosition, rook.Color) {
				newBoard := board.DeepCopy()
				newBoard.MovePiece(position, newPosition)
				possibleBoards = append(possibleBoards, newBoard)
				break
			} else {
				break
			}
		}
	}

	// Filter boards if inCheck is true
	if inCheck {
		possibleBoards = filterBoardsInCheck(possibleBoards, rook.Color)
	}

	return possibleBoards
}

func (rook Rook) GetColor() bool {
	return rook.Color
}

func (rook Rook) ToString() string {
	if rook.Color {
		return " W R "
	} else {
		return " B R "
	}
}
