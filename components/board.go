package components

import (
	"encoding/json"
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

type ChessPieceJSON struct {
	Type  string
	Color bool
}

func (cb *ChessBoard) MarshalJSON() ([]byte, error) {
	board := [8][8]ChessPieceJSON{}
	for y, row := range cb.Board {
		for x, piece := range row {
			if piece != nil {
				board[y][x] = ChessPieceJSON{
					Type:  piece.ToString(),
					Color: piece.GetColor(),
				}
			}
		}
	}
	return json.Marshal(struct {
		Board    [8][8]ChessPieceJSON
		Score    int
		NextTurn bool
	}{
		Board:    board,
		Score:    cb.Score,
		NextTurn: cb.NextTurn,
	})
}

func (cb *ChessBoard) UnmarshalJSON(data []byte) error {
	aux := struct {
		Board    [8][8]ChessPieceJSON
		Score    int
		NextTurn bool
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	for y, row := range aux.Board {
		for x, pieceJSON := range row {
			switch pieceJSON.Type {
			case " W P ", " B P ":
				cb.Board[y][x] = &Pawn{Color: pieceJSON.Color}
			case " W R ", " B R ":
				cb.Board[y][x] = &Rook{Color: pieceJSON.Color}
			case " W N ", " B N ":
				cb.Board[y][x] = &Knight{Color: pieceJSON.Color}
			case " W B ", " B B ":
				cb.Board[y][x] = &Bishop{Color: pieceJSON.Color}
			case " W Q ", " B Q ":
				cb.Board[y][x] = &Queen{Color: pieceJSON.Color}
			case " W K ", " B K ":
				cb.Board[y][x] = &King{Color: pieceJSON.Color}
			default:
				cb.Board[y][x] = nil
			}
		}
	}

	cb.Score = aux.Score
	cb.NextTurn = aux.NextTurn
	return nil
}

type AttackCache struct {
	Positions map[Coordinates]bool
	Computed  bool
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
	if position.X < 0 || position.X >= 8 || position.Y < 0 || position.Y >= 8 {
		// fmt.Println("out of bounds")
		return false
	}
	// fmt.Println("Is Empty (", position.X, ")", "(", position.Y, "): ", cb.Board[position.Y][position.X])
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
	// fmt.Println("FROM: ", from)
	// fmt.Println("TO: ", to)

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

// Compute attacks directly without recursion
func (cb *ChessBoard) ComputeAttacks(color bool) map[Coordinates]bool {
	attacks := make(map[Coordinates]bool)

	// For each piece on the board
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			piece := cb.Board[y][x]
			if piece == nil || piece.GetColor() != color {
				continue
			}

			// Handle different piece types directly
			switch piece.ToString() {
			case " W P ", " B P ":
				// Pawns attack diagonally
				direction := 1
				if !piece.GetColor() {
					direction = -1
				}
				if x > 0 {
					attacks[Coordinates{X: x - 1, Y: y + direction}] = true
				}
				if x < 7 {
					attacks[Coordinates{X: x + 1, Y: y + direction}] = true
				}

			case " W N ", " B N ":
				// Knight moves
				knightMoves := []Coordinates{
					{X: x + 2, Y: y + 1}, {X: x + 2, Y: y - 1},
					{X: x - 2, Y: y + 1}, {X: x - 2, Y: y - 1},
					{X: x + 1, Y: y + 2}, {X: x + 1, Y: y - 2},
					{X: x - 1, Y: y + 2}, {X: x - 1, Y: y - 2},
				}
				for _, move := range knightMoves {
					if cb.IsWithinBounds(move) {
						attacks[move] = true
					}
				}

			case " W K ", " B K ":
				// King moves (one square in any direction)
				for dx := -1; dx <= 1; dx++ {
					for dy := -1; dy <= 1; dy++ {
						if dx == 0 && dy == 0 {
							continue
						}
						newPos := Coordinates{X: x + dx, Y: y + dy}
						if cb.IsWithinBounds(newPos) {
							attacks[newPos] = true
						}
					}
				}

			case " W B ", " B B ", " W Q ", " B Q ":
				// Bishop/Queen diagonal moves
				for i := 1; i < 8; i++ {
					directions := []Coordinates{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
					for _, dir := range directions {
						newPos := Coordinates{X: x + i*dir.X, Y: y + i*dir.Y}
						if !cb.IsWithinBounds(newPos) {
							break
						}
						attacks[newPos] = true
						if !cb.IsEmpty(newPos) {
							break
						}
					}
				}
				if piece.ToString() == " W B " || piece.ToString() == " B B " {
					break
				}
				fallthrough // Continue to rook moves for queen

			case " W R ", " B R ":
				// Rook/Queen straight moves
				for i := 1; i < 8; i++ {
					directions := []Coordinates{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
					for _, dir := range directions {
						newPos := Coordinates{X: x + i*dir.X, Y: y + i*dir.Y}
						if !cb.IsWithinBounds(newPos) {
							break
						}
						attacks[newPos] = true
						if !cb.IsEmpty(newPos) {
							break
						}
					}
				}
			}
		}
	}
	return attacks
}

func (cb *ChessBoard) WouldLeaveKingInCheck(color bool, cache *AttackCache, _ int) bool {
	// Find king position
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

	// Use cached attacks if available
	if cache != nil && cache.Computed {
		return cache.Positions[kingPosition]
	}

	// Compute opponent attacks directly
	attacks := cb.ComputeAttacks(!color)

	// Store in cache if provided
	if cache != nil {
		cache.Positions = attacks
		cache.Computed = true
	}

	return attacks[kingPosition]
}

func (cb *ChessBoard) IsWithinBounds(position Coordinates) bool {
	return position.X >= 0 && position.X < 8 && position.Y >= 0 && position.Y < 8
}
