package components

import (
	"fmt"
)

type Coordinates struct {
	X int
	Y int
}

type ChessBoard struct {
	Board    [8][8]ChessPiece
	Score    int  // + white / - black
	NextTurn bool // true = white, flase = black
}

// Temp for Debug -> remake for more efficient storage later
func (chessBoard ChessBoard) ToString() string {
	fmt.Println("  ------------------------------------------")
	for rNum, row := range chessBoard.Board {
		fmt.Print(rNum, "-|")
		for _, piece := range row {
			if piece != nil {
				fmt.Print(piece.ToString())
			} else {
				fmt.Print("     ") // Print empty space for nil pieces
			}
		}
		fmt.Println("|\n")
	}
	fmt.Println("  ------------------------------------------")
	fmt.Println("     A    B    C    D    E    F    G    H  ")
	return ""
}

func (cb *ChessBoard) IsEmpty(position Coordinates) bool {
	// Check if the position is within the bounds of the board
	if position.X < 0 || position.X >= 8 || position.Y < 0 || position.Y >= 8 {
		fmt.Println("out of bounds")
		return false
	}
	fmt.Println("Is Empty (", position.X, ")", "(", position.Y, "): ", cb.Board[position.Y][position.X])

	// Return true if the position is nil (empty)
	return cb.Board[position.Y][position.X] == nil
}

// DeepCopy creates a deep copy of a ChessBoard instance, ensuring independence from the original.
func (cb *ChessBoard) DeepCopy() *ChessBoard {
	// Create a new ChessBoard instance
	newBoard := ChessBoard{
		Score:    cb.Score,
		NextTurn: cb.NextTurn,
	}

	// Deep copy each ChessPiece in the Board
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			newBoard.Board[i][j] = cb.Board[i][j] // Assumes ChessPiece itself does not have pointers
		}
	}

	return &newBoard
}

func (cb *ChessBoard) MovePiece(from, to Coordinates) {
	// Check if the 'from' position is within bounds and has a piece
	if from.X < 0 || from.X >= 8 || from.Y < 0 || from.Y >= 8 || cb.Board[from.X][from.Y] == nil {
		return
	}
	fmt.Println("FROM: ", from)
	fmt.Println("TO: ", to)

	// Move the piece to the new position
	cb.Board[to.Y][to.X] = cb.Board[from.Y][from.X]
	// Set the original position to nil (empty)
	cb.Board[from.Y][from.X] = nil
}

func (cb *ChessBoard) IsEnemy(position Coordinates, color bool) bool {
	// Check if the position is within the bounds of the board
	if position.X < 0 || position.X >= 8 || position.Y < 0 || position.Y >= 8 {
		return false
	}

	// Get the piece at the given position
	piece := cb.Board[position.X][position.Y]

	// Check if there is a piece and if it is an enemy
	return piece != nil && piece.GetColor() != color
}

func (cb *ChessBoard) WouldLeaveKingInCheck(color bool) bool {
	// Find the king's position
	var kingPosition Coordinates
	found := false
	for x := 0; x < 8 && !found; x++ {
		for y := 0; y < 8 && !found; y++ {
			piece := cb.Board[y][x]
			if piece != nil && piece.GetColor() == color && piece.ToString() == " K " {
				kingPosition = Coordinates{X: x, Y: y}
				found = true
			}
		}
	}

	// Check if any opponent piece can attack the king's position
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece := cb.Board[x][y]
			if piece != nil && piece.GetColor() != color {
				// Assuming each piece has a method to get possible attack positions
				possibleAttacks := piece.GetPossibleMoves(*cb, Coordinates{X: x, Y: y}, false)
				for _, attackBoard := range possibleAttacks {
					if attackBoard.Board[kingPosition.X][kingPosition.Y] != nil &&
						attackBoard.Board[kingPosition.X][kingPosition.Y].ToString() == " K " {
						return true
					}
				}
			}
		}
	}

	return false
}

func (cb *ChessBoard) IsWithinBounds(position Coordinates) bool {
	// Check if the position is within the bounds of the board
	return position.X >= 0 && position.X < 8 && position.Y >= 0 && position.Y < 8
}
