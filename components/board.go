package components

import "fmt"

type Coordinates struct {
	x int16
	y int16
}

type ChessBoard struct {
	Board    [8][8]ChessPiece
	Score    int16 // + white / - black
	NextTurn bool  // true = white, flase = black
}

// Temp for Debug -> remake for more efficient storage later
func (chessBoard ChessBoard) ToString() string {
	fmt.Println("  ------------------------------------------")
	for rNum, row := range chessBoard.Board {
		fmt.Print(rNum, "-|")
		for _, piece := range row {
			fmt.Print(piece.ToString())
		}
		fmt.Println("|\n")
	}
	fmt.Println("  ------------------------------------------")
	fmt.Println("     A    B    C    D    E    F    G    H  ")
	return ""
}
