package main

import (
	"fmt"

	"github.com/LIAMBB/chess-compute/components"
	"github.com/davecgh/go-spew/spew"
)

type BoardRouteNode struct {
	CurrentState *components.ChessBoard
	NextStates   []*components.ChessBoard
}

func simulateGames(node *BoardRouteNode, depth int, maxDepth int) {

	spew.Dump(node.CurrentState.Board[0][3])
	possibleStates := node.CurrentState.Board[0][3].GetPossibleMoves(*node.CurrentState, components.Coordinates{X: 3, Y: 0}, false)
	for _, b := range possibleStates {
		b.ToString()
	}
}

func main() {
	startingBoard := initGame()
	rootNode := &BoardRouteNode{CurrentState: &startingBoard, NextStates: make([]*components.ChessBoard, 0)}
	maxDepth := 3 // Set your desired maximum depth here
	simulateGames(rootNode, 0, maxDepth)

	// Example output to verify the simulation
	fmt.Println("Simulation complete. Number of states generated:", len(rootNode.NextStates))
}

func initGame() components.ChessBoard {
	board := [8][8]components.ChessPiece{
		{components.Rook{Color: true}, components.Knight{Color: true}, components.Bishop{Color: true}, components.Queen{Color: true}, components.King{Color: true}, components.Bishop{Color: true}, components.Knight{Color: true}, components.Rook{Color: true}},
		// {components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, nil, nil, nil, components.Pawn{Color: true}, components.Pawn{Color: true}},
		// {components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}},
		{components.Pawn{Color: true}, components.Pawn{Color: true}, nil, nil, nil, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}},
		// {nil, nil, nil, nil, nil, nil, nil, nil},
		{components.Pawn{Color: false}, nil, nil, nil, nil, nil, nil, nil},
		{nil, nil, nil, nil, nil, nil, nil, nil},
		{nil, nil, nil, nil, nil, nil, nil, nil},
		{nil, nil, nil, nil, nil, nil, nil, nil},
		{components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}},
		{components.Rook{Color: false}, components.Knight{Color: false}, components.Bishop{Color: false}, components.Queen{Color: false}, components.King{Color: false}, components.Bishop{Color: false}, components.Knight{Color: false}, components.Rook{Color: false}},
	}
	// components.Rook{Color: false},

	return components.ChessBoard{
		Score:    0,
		NextTurn: true,
		Board:    board,
	}
}
