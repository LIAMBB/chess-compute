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
	NextTurn bool // true = white, false = black
}

type AttackCache map[Coordinates]bool

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
	if position.X < 0 || position.X >= 8 || position.Y < 0 || position.Y >= 8 {
		fmt.Println("out of bounds")
		return false
	}
	fmt.Println("Is Empty (", position.X, ")", "(", position.Y, "): ", cb.Board[position.Y][position.X])
	return cb.Board[position.Y][position.X] == nil
}

func (cb *ChessBoard) DeepCopy() *ChessBoard {
	newBoard := ChessBoard{
		Score:    cb.Score,
		NextTurn: cb.NextTurn,
	}

	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			newBoard.Board[i][j] = cb.Board[i][j]
		}
	}

	return &newBoard
}

func (cb *ChessBoard) MovePiece(from, to Coordinates) {
	if from.X < 0 || from.X >= 8 || from.Y < 0 || from.Y >= 8 || cb.Board[from.Y][from.X] == nil {
		return
	}
	fmt.Println("FROM: ", from)
	fmt.Println("TO: ", to)

	cb.Board[to.Y][to.X] = cb.Board[from.Y][from.X]
	cb.Board[from.Y][from.X] = nil
}

func (cb *ChessBoard) IsEnemy(position Coordinates, color bool) bool {
	if position.X < 0 || position.X >= 8 || position.Y < 0 || position.Y >= 8 {
		return false
	}

	piece := cb.Board[position.Y][position.X]
	return piece != nil && piece.GetColor() != color
}

func (cb *ChessBoard) WouldLeaveKingInCheck(color bool, cache AttackCache) bool {
	if cache == nil {
		cache = cb.ComputeOpponentAttacks(!color)
	}

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

	return cache[kingPosition]
}

func (cb *ChessBoard) ComputeOpponentAttacks(opponentColor bool) AttackCache {
	cache := make(AttackCache)

	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			piece := cb.Board[y][x]
			if piece != nil && piece.GetColor() == opponentColor {
				possibleMoves := piece.GetPossibleMoves(*cb, Coordinates{X: x, Y: y}, false)
				for _, board := range possibleMoves {
					for i := 0; i < 8; i++ {
						for j := 0; j < 8; j++ {
							if board.Board[i][j] != nil && board.Board[i][j].GetColor() == opponentColor {
								cache[Coordinates{X: i, Y: j}] = true
							}
						}
					}
				}
			}
		}
	}

	return cache
}

func (cb *ChessBoard) IsWithinBounds(position Coordinates) bool {
	return position.X >= 0 && position.X < 8 && position.Y >= 0 && position.Y < 8
}
