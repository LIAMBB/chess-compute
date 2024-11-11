package components

import "fmt"

type Pawn struct {
	Color bool // white = true, black = false
}

func (pawn Pawn) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	var possibleBoards []*ChessBoard

	// Determine direction based on color
	direction := 1
	if !pawn.Color {
		direction = -1
	}
	fmt.Println("Direction: ", direction)

	// Forward 1
	oneStepForward := Coordinates{X: position.X, Y: position.Y + direction}

	fmt.Println("FROM: ", Coordinates{X: position.X, Y: position.Y})
	fmt.Println("TO: ", oneStepForward)
	if board.IsEmpty(oneStepForward) {
		newBoard := board.DeepCopy()
		newBoard.MovePiece(position, oneStepForward)
		possibleBoards = append(possibleBoards, newBoard)

	}

	// Forward 2 (only if first move)
	if (pawn.Color && position.Y == 1) || (!pawn.Color && position.Y == 6) {
		twoStepsForward := Coordinates{X: position.X, Y: position.Y + 2*direction}

		fmt.Println("FROM: ", Coordinates{X: position.X, Y: position.Y})
		fmt.Println("TO: ", twoStepsForward)
		if board.IsEmpty(twoStepsForward) {
			newBoard := board.DeepCopy()
			newBoard.MovePiece(position, twoStepsForward)
			possibleBoards = append(possibleBoards, newBoard)
		}
	}

	// Capture diagonally left
	diagonalLeft := Coordinates{X: position.X - 1, Y: position.Y + direction}
	if board.IsEnemy(diagonalLeft, pawn.Color) {
		newBoard := board.DeepCopy()
		newBoard.MovePiece(position, diagonalLeft)
		possibleBoards = append(possibleBoards, newBoard)
	}

	// Capture diagonally right
	diagonalRight := Coordinates{X: position.X + 1, Y: position.Y + direction}
	if board.IsEnemy(diagonalRight, pawn.Color) {
		newBoard := board.DeepCopy()
		newBoard.MovePiece(position, diagonalRight)
		possibleBoards = append(possibleBoards, newBoard)
	}

	// TODO: handle En-Passant scenario

	// Filter boards if inCheck is true
	if inCheck {
		possibleBoards = filterBoardsInCheck(possibleBoards, pawn.Color)
	}

	return possibleBoards
}

// Helper function to filter boards that would leave the king in check
func filterBoardsInCheck(boards []*ChessBoard, color bool) []*ChessBoard {
	var validBoards []*ChessBoard
	for _, board := range boards {
		if !board.WouldLeaveKingInCheck(color) {
			validBoards = append(validBoards, board)
		}
	}
	return validBoards
}

func (pawn Pawn) GetColor() bool {
	return pawn.Color
}

func (pawn Pawn) ToString() string {
	if pawn.Color {
		return " W P "
	} else {
		return " B P "
	}
}
