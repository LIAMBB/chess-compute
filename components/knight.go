package components

import "fmt"

type Knight struct {
	Color bool // white = true, black = false
}

func (knight Knight) GetPossibleMoves(board ChessBoard, position Coordinates, inCheck bool) []*ChessBoard {
	var possibleBoards []*ChessBoard
	// Vertical (y +/- 2) (x +/- 1)
	// Horizontal (y +/- 1) (x +/- 2)
	// Define all possible moves for a knight
	fmt.Println("Getting possible moves for knight")
	moves := []Coordinates{
		{X: position.X + 2, Y: position.Y + 1},
		{X: position.X + 2, Y: position.Y - 1},
		{X: position.X - 2, Y: position.Y + 1},
		{X: position.X - 2, Y: position.Y - 1},
		{X: position.X + 1, Y: position.Y + 2},
		{X: position.X + 1, Y: position.Y - 2},
		{X: position.X - 1, Y: position.Y + 2},
		{X: position.X - 1, Y: position.Y - 2},
	}

	for _, move := range moves {
		if board.IsWithinBounds(move) && (board.IsEmpty(move) || board.IsEnemy(move, knight.Color)) {
			fmt.Println("Move is valid")
			newBoard := board.DeepCopy()
			newBoard.MovePiece(position, move)
			possibleBoards = append(possibleBoards, newBoard)
		}
	}

	// Filter boards if inCheck is true
	if inCheck {
		possibleBoards = filterBoardsInCheck(possibleBoards, knight.Color)
	}

	return possibleBoards
}

func (knight Knight) GetColor() bool {
	return knight.Color
}

func (knight Knight) ToString() string {
	if knight.Color {
		return " W N "
	} else {
		return " B N "
	}
}
