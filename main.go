package main

import (
	"fmt"

	"github.com/LIAMBB/chess-compute/components"
)

var simulatedBoardStates = make(map[string]*components.ChessBoard)
var gameSimulation BoardRouteNode
var dumpedStates = make(map[string]bool)

type BoardRouteNode struct {
	CurrentState *components.ChessBoard
	NextStates   []*components.ChessBoard
}

func main() {
	fmt.Println("Hello World")
	gameBoard := initGame()
	fmt.Println(gameBoard.ToString())
	// simulateGames()
}

func initGame() components.ChessBoard {
	board := [8][8]components.ChessPiece{
		{components.Rook{Color: true}, components.Knight{Color: true}, components.Bishop{Color: true}, components.Queen{Color: true}, components.Knight{Color: true}, components.Bishop{Color: true}, components.Knight{Color: true}, components.Rook{Color: true}},
		{components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}},
		{components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}},
		{components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}},
		{components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}},
		{components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}, components.EmptySpace{}},
		{components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}},
		{components.Rook{Color: false}, components.Knight{Color: false}, components.Bishop{Color: false}, components.Queen{Color: false}, components.Knight{Color: false}, components.Bishop{Color: false}, components.Knight{Color: false}, components.Rook{Color: false}},
	}
	// components.Rook{Color: false},

	return components.ChessBoard{
		Score:    0,
		NextTurn: true,
		Board:    board,
	}
}

// func simulateGames() {
// 	startingBoard := initGame()
// 	rootNode := BoardRouteNode{CurrentState: &startingBoard, NextStates: make([]*components.ChessBoard, 0)}
// 	for i := 0; i < 100; i++ {
// 		current
// 		for x, row := range rootNode.CurrentState.Board {
// 			for y, tile := range row {
// 				if !tile.IsEmpty() && tile.GetColor() ==  {

// 				}
// 			}
// 		}
// 	}
// }
