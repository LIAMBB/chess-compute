package components

type Bishop struct {
	Color bool // white = true, black = false
}

func (bishop Bishop) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	var possibleBoards []*ChessBoard

	// Define directions for diagonal movement
	directions := []Coordinates{
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
			} else if board.IsEnemy(newPosition, bishop.Color) {
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
		possibleBoards = filterBoardsInCheck(possibleBoards, bishop.Color)
	}

	return possibleBoards
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
