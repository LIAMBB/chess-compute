package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"syscall"

	"github.com/LIAMBB/chess-compute/components"
	_ "github.com/mattn/go-sqlite3"
)

type BoardRouteNode struct {
	StateID      int
	NextStateIDs []int
}

func simulateGamesWithSQLite(ctx context.Context, rootNode *BoardRouteNode, db *sql.DB, maxDepth int) {
	// Create excess queue table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS excess_queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		state_id INTEGER NOT NULL,
		depth INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Println("Error creating excess queue table:", err)
		return
	}

	queue := []*BoardRouteNode{rootNode}
	currentDepth := 0
	batchSize := 100
	queueLimit := 10000
	totalProcessed := 0
	statesAtDepth := make(map[int]int)

	for len(queue) > 0 && currentDepth < maxDepth {
		select {
		case <-ctx.Done():
			fmt.Println("Simulation stopping...")
			return
		default:
			levelSize := len(queue)
			fmt.Printf("Processing depth %d with %d nodes\n", currentDepth, levelSize)

			var newQueue []*BoardRouteNode
			results := make(chan []*BoardRouteNode, levelSize)
			errChan := make(chan error, levelSize)

			// Create a worker pool
			workerCount := runtime.NumCPU()
			workChan := make(chan struct {
				start, end int
				parentNode *BoardRouteNode
			}, levelSize)

			// Start worker pool
			var wg sync.WaitGroup
			for i := 0; i < workerCount; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for work := range workChan {
						select {
						case <-ctx.Done():
							return
						default:
							var localResults []*BoardRouteNode

							for j := work.start; j < work.end; j++ {
								node := queue[j]
								currentBoard := getBoardStateByID(db, node.StateID)
								if currentBoard == nil {
									errChan <- fmt.Errorf("nil board state for ID %d", node.StateID)
									continue
								}

								moveCount := 0
								for y, row := range currentBoard.Board {
									for x, piece := range row {
										if piece != nil && piece.GetColor() == currentBoard.NextTurn {
											possibleMoves := piece.GetPossibleMoves(*currentBoard, components.Coordinates{X: x, Y: y}, false)
											moveCount += len(possibleMoves)
											for _, newBoard := range possibleMoves {
												newBoard.NextTurn = !currentBoard.NextTurn
												newStateID := storeBoardState(db, newBoard)
												if newStateID == -1 {
													errChan <- fmt.Errorf("failed to store board state")
													continue
												}
												newNode := &BoardRouteNode{
													StateID:      newStateID,
													NextStateIDs: []int{},
												}
												work.parentNode.NextStateIDs = append(work.parentNode.NextStateIDs, newStateID)
												localResults = append(localResults, newNode)
											}
										}
									}
								}
								if moveCount == 0 {
									fmt.Printf("Warning: No moves generated for board at depth %d\n", currentDepth)
								}
							}

							select {
							case <-ctx.Done():
								return
							case results <- localResults:
							}
						}
					}
				}()
			}

			// Distribute work
			for i := 0; i < levelSize; i += batchSize {
				end := i + batchSize
				if end > levelSize {
					end = levelSize
				}

				select {
				case <-ctx.Done():
					close(workChan)
					return
				case workChan <- struct {
					start, end int
					parentNode *BoardRouteNode
				}{i, end, queue[i]}:
				}
			}
			close(workChan)

			// Wait for all workers to finish
			go func() {
				wg.Wait()
				close(results)
				close(errChan)
			}()

			// Collect results
			for result := range results {
				newQueue = append(newQueue, result...)
			}

			// Check for errors
			for err := range errChan {
				if err != nil {
					fmt.Printf("Error processing states: %v\n", err)
				}
			}

			statesAtDepth[currentDepth] = len(newQueue)

			// Store excess states in database
			if len(newQueue) > queueLimit {
				fmt.Printf("Storing %d excess states in database at depth %d\n", len(newQueue)-queueLimit, currentDepth)
				excessQueue := newQueue[queueLimit:]
				newQueue = newQueue[:queueLimit]

				// Store excess states in database in batches
				batchSize := 1000
				for i := 0; i < len(excessQueue); i += batchSize {
					end := i + batchSize
					if end > len(excessQueue) {
						end = len(excessQueue)
					}

					tx, err := db.Begin()
					if err != nil {
						fmt.Println("Error starting transaction for excess queue:", err)
						continue
					}

					stmt, err := tx.Prepare("INSERT INTO excess_queue (state_id, depth) VALUES (?, ?)")
					if err != nil {
						fmt.Println("Error preparing excess queue statement:", err)
						tx.Rollback()
						continue
					}

					for _, node := range excessQueue[i:end] {
						_, err := stmt.Exec(node.StateID, currentDepth)
						if err != nil {
							fmt.Printf("Error storing excess state %d: %v\n", node.StateID, err)
						}
					}

					stmt.Close()
					if err := tx.Commit(); err != nil {
						fmt.Println("Error committing excess queue transaction:", err)
					}
				}
			}

			// Process all stored states at current depth before moving on
			for {
				// Count remaining states at current depth
				var remainingCount int
				err := db.QueryRow("SELECT COUNT(*) FROM excess_queue WHERE depth = ?", currentDepth).Scan(&remainingCount)
				if err != nil {
					fmt.Println("Error counting remaining states:", err)
					break
				}

				if remainingCount == 0 {
					break
				}

				fmt.Printf("Processing %d remaining states at depth %d\n", remainingCount, currentDepth)

				// Fetch next batch of stored states
				rows, err := db.Query("SELECT state_id FROM excess_queue WHERE depth = ? LIMIT ?", currentDepth, queueLimit)
				if err != nil {
					fmt.Println("Error querying excess queue:", err)
					break
				}

				var storedStates []*BoardRouteNode
				var processedIDs []int
				for rows.Next() {
					var stateID int
					err := rows.Scan(&stateID)
					if err != nil {
						fmt.Println("Error scanning excess queue row:", err)
						continue
					}
					storedStates = append(storedStates, &BoardRouteNode{StateID: stateID, NextStateIDs: []int{}})
					processedIDs = append(processedIDs, stateID)
				}
				rows.Close()

				if len(storedStates) == 0 {
					break
				}

				// Delete processed states
				if len(processedIDs) > 0 {
					query := "DELETE FROM excess_queue WHERE depth = ? AND state_id IN ("
					params := []interface{}{currentDepth}
					for i, id := range processedIDs {
						if i > 0 {
							query += ","
						}
						query += "?"
						params = append(params, id)
					}
					query += ")"

					_, err = db.Exec(query, params...)
					if err != nil {
						fmt.Println("Error deleting processed excess states:", err)
					}
				}

				// Process stored states and add results to newQueue instead of replacing queue
				tempQueue := storedStates
				var tempResults []*BoardRouteNode

				// Process the stored states similar to main loop
				for i := 0; i < len(tempQueue); i += batchSize {
					end := i + batchSize
					if end > len(tempQueue) {
						end = len(tempQueue)
					}

					for j := i; j < end; j++ {
						node := tempQueue[j]
						currentBoard := getBoardStateByID(db, node.StateID)
						if currentBoard == nil {
							continue
						}

						for y, row := range currentBoard.Board {
							for x, piece := range row {
								if piece != nil && piece.GetColor() == currentBoard.NextTurn {
									possibleMoves := piece.GetPossibleMoves(*currentBoard, components.Coordinates{X: x, Y: y}, false)
									for _, newBoard := range possibleMoves {
										newBoard.NextTurn = !currentBoard.NextTurn
										newStateID := storeBoardState(db, newBoard)
										newNode := &BoardRouteNode{
											StateID:      newStateID,
											NextStateIDs: []int{},
										}
										tempResults = append(tempResults, newNode)
									}
								}
							}
						}
					}
				}

				// Add processed stored states to newQueue
				newQueue = append(newQueue, tempResults...)
				totalProcessed += len(tempQueue)
			}

			fmt.Printf("Depth %d complete. Generated %d new states. Total processed: %d\n",
				currentDepth, len(newQueue), totalProcessed)
			fmt.Printf("States at each depth: %v\n", statesAtDepth)

			// Verify no states were left behind
			var remainingCount int
			err = db.QueryRow("SELECT COUNT(*) FROM excess_queue WHERE depth = ?", currentDepth).Scan(&remainingCount)
			if err != nil {
				fmt.Println("Error checking for remaining states:", err)
			} else if remainingCount > 0 {
				fmt.Printf("Warning: %d states were left unprocessed at depth %d\n", remainingCount, currentDepth)
			}

			queue = newQueue
			currentDepth++
		}
	}
}

func storeBoardState(db *sql.DB, board *components.ChessBoard) int {
	data, err := json.Marshal(board)
	if err != nil {
		fmt.Println("Error marshaling state:", err)
		return -1
	}

	result, err := db.Exec("INSERT INTO board_states (state) VALUES (?)", data)
	if err != nil {
		fmt.Println("Error inserting state into database:", err)
		return -1
	}

	id, err := result.LastInsertId()
	if err != nil {
		fmt.Println("Error getting last insert ID:", err)
		return -1
	}

	return int(id)
}

func getBoardStateByID(db *sql.DB, id int) *components.ChessBoard {
	var data string
	err := db.QueryRow("SELECT state FROM board_states WHERE id = ?", id).Scan(&data)
	if err != nil {
		fmt.Println("Error querying state from database:", err)
		return nil
	}

	var board components.ChessBoard
	err = json.Unmarshal([]byte(data), &board)
	if err != nil {
		fmt.Println("Error unmarshaling state:", err)
		return nil
	}

	return &board
}

func storeBoardStatesBatch(db *sql.DB, boards []*components.ChessBoard) []int {
	tx, err := db.Begin()
	if err != nil {
		fmt.Println("Error starting transaction:", err)
		return nil
	}

	stmt, err := tx.Prepare("INSERT INTO board_states (state) VALUES (?)")
	if err != nil {
		fmt.Println("Error preparing statement:", err)
		return nil
	}
	defer stmt.Close()

	var ids []int
	for _, board := range boards {
		data, err := json.Marshal(board)
		if err != nil {
			fmt.Println("Error marshaling state:", err)
			continue
		}

		result, err := stmt.Exec(data)
		if err != nil {
			fmt.Println("Error executing statement:", err)
			continue
		}

		id, err := result.LastInsertId()
		if err != nil {
			fmt.Println("Error getting last insert ID:", err)
			continue
		}

		ids = append(ids, int(id))
	}

	if err := tx.Commit(); err != nil {
		fmt.Println("Error committing transaction:", err)
		return nil
	}

	return ids
}

func main() {
	db, err := sql.Open("sqlite3", "./chess.db")
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	// Enable WAL mode and optimize SQLite settings
	_, err = db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA cache_size=10000;
		PRAGMA temp_store=MEMORY;
		PRAGMA mmap_size=30000000000;
	`)
	if err != nil {
		fmt.Println("Error setting SQLite pragmas:", err)
		return
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS board_states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		state TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Println("Error creating table:", err)
		return
	}

	startingBoard := initGame()
	rootStateID := storeBoardState(db, &startingBoard)
	rootNode := &BoardRouteNode{StateID: rootStateID, NextStateIDs: []int{}}
	maxDepth := 7 // Set your desired maximum depth here

	// Set up context and signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a done channel to signal completion
	done := make(chan bool, 1)

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create a channel to indicate computation is complete
	computationDone := make(chan bool, 1)

	// Handle shutdown in a separate goroutine
	go func() {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, shutting down...")
			cancel()
			// Wait for cleanup
			<-done
			fmt.Println("Cleanup complete, exiting...")
			os.Exit(0)
		case <-computationDone:
			// Stop listening for signals once computation is complete
			signal.Stop(sigChan)
			return
		}
	}()

	// Run simulation in a separate goroutine
	go func() {
		simulateGamesWithSQLite(ctx, rootNode, db, maxDepth)
		done <- true
		computationDone <- true
	}()

	// Wait for simulation to complete or context to be cancelled
	select {
	case <-ctx.Done():
		fmt.Println("Simulation cancelled")
	case <-done:
		fmt.Println("Simulation complete")
		// Start CLI only if simulation completed normally
		traverseTree(rootNode, db)
	}
}

func initGame() components.ChessBoard {
	board := [8][8]components.ChessPiece{
		{components.Rook{Color: true}, components.Knight{Color: true}, components.Bishop{Color: true}, components.Queen{Color: true}, components.King{Color: true}, components.Bishop{Color: true}, components.Knight{Color: true}, components.Rook{Color: true}},
		{components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}, components.Pawn{Color: true}},
		{nil, nil, nil, nil, nil, nil, nil, nil},
		{nil, nil, nil, nil, nil, nil, nil, nil},
		{nil, nil, nil, nil, nil, nil, nil, nil},
		{nil, nil, nil, nil, nil, nil, nil, nil},
		{components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}, components.Pawn{Color: false}},
		{components.Rook{Color: false}, components.Knight{Color: false}, components.Bishop{Color: false}, components.Queen{Color: false}, components.King{Color: false}, components.Bishop{Color: false}, components.Knight{Color: false}, components.Rook{Color: false}},
	}

	return components.ChessBoard{
		Score:    0,
		NextTurn: true, // Start with white's turn
		Board:    board,
	}
}

func traverseTree(node *BoardRouteNode, db *sql.DB) {
	reader := bufio.NewReader(os.Stdin)
	currentNode := node

	for {
		fmt.Println("Current Board State:")
		board := getBoardStateByID(db, currentNode.StateID)
		board.ToString()

		if len(currentNode.NextStateIDs) == 0 {
			fmt.Println("No further moves available.")
			break
		}

		fmt.Println("Available Moves:")
		for i, stateID := range currentNode.NextStateIDs {
			fmt.Printf("%d: Move to state ID %d\n", i, stateID)
		}

		fmt.Print("Enter the number of the move to make, or 'b' to go back: ")
		input, _ := reader.ReadString('\n')
		input = input[:len(input)-1] // Remove newline character

		if input == "b" {
			// Implement backtracking logic if needed
			fmt.Println("Backtracking is not implemented in this version.")
			continue
		}

		moveIndex, err := strconv.Atoi(input)
		if err != nil || moveIndex < 0 || moveIndex >= len(currentNode.NextStateIDs) {
			fmt.Println("Invalid input. Please enter a valid move number.")
			continue
		}

		currentNode = &BoardRouteNode{StateID: currentNode.NextStateIDs[moveIndex], NextStateIDs: []int{}}
	}
}
