package main

import "fmt"

type Coordinates struct {
	x int16
	y int16
}
type ChessPiece interface {
	GetPossibleMoves(board ChessBoard, position Coordinates) []Coordinates
	GetColor() bool
	IsEmpty() bool
}

type ChessBoard struct {
	Board    [8][8]ChessPiece
	Score    int16 // + white / - black
	NextTurn bool  // true = white, flase = black
}

func main() {
	fmt.Println("Hello World")
}
