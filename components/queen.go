package components

type Queen struct {
	Color bool // white = true, black = false
}

func (queen Queen) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	var possibleBoards []*ChessBoard

	// Define directions for both rook and bishop movements
	directions := []Coordinates{
		// Rook-like movements
		{X: 1, Y: 0},  // Right
		{X: -1, Y: 0}, // Left
		{X: 0, Y: 1},  // Down
		{X: 0, Y: -1}, // Up
		// Bishop-like movements
		{X: 1, Y: 1},   // Down-right
		{X: 1, Y: -1},  // Up-right
		{X: -1, Y: 1},  // Down-left
		{X: -1, Y: -1}, // Up-left
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
			} else if board.IsEnemy(newPosition, queen.Color) {
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
		possibleBoards = filterBoardsInCheck(possibleBoards, queen.Color)
	}

	return possibleBoards
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
