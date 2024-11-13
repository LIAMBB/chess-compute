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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/LIAMBB/chess-compute/components"
	_ "github.com/mattn/go-sqlite3"
)

type BoardRouteNode struct {
	StateID      int
	NextStateIDs []int
}

func checkDatabaseSize(db *sql.DB, maxSizeBytes int64) (bool, error) {
	// Force a checkpoint to ensure WAL is considered
	_, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	if err != nil {
		return false, fmt.Errorf("checkpoint error: %v", err)
	}

	var size, walSize int64

	// Get main DB size
	err = db.QueryRow("SELECT page_count * page_size FROM pragma_page_count, pragma_page_size").Scan(&size)
	if err != nil {
		return false, err
	}

	// Get WAL size
	err = db.QueryRow("SELECT SUM(s) FROM (SELECT size as s FROM pragma_wal_checkpoint)").Scan(&walSize)
	if err != nil {
		// If error, just use main DB size
		walSize = 0
	}

	totalSize := size + walSize

	// Use a safety margin (90% of max size)
	margin := int64(float64(maxSizeBytes) * 0.9)
	exceeded := totalSize >= margin

	if exceeded {
		fmt.Printf("Database size check: Total %.2f GB (DB: %.2f GB, WAL: %.2f GB), Limit: %.2f GB\n",
			float64(totalSize)/1024/1024/1024,
			float64(size)/1024/1024/1024,
			float64(walSize)/1024/1024/1024,
			float64(maxSizeBytes)/1024/1024/1024)
	}

	return exceeded, nil
}

func simulateGamesWithSQLite(ctx context.Context, rootNode *BoardRouteNode, db *sql.DB, maxDepth int, maxSizeBytes int64) {
	// Create a new context that can be cancelled when DB is full
	dbCtx, dbCancel := context.WithCancel(ctx)
	defer dbCancel() // Ensure cleanup

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

	// Add transaction for storing node relations
	tx, err := db.Begin()
	if err != nil {
		fmt.Println("Error starting transaction:", err)
		return
	}

	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO node_relations (parent_id, child_id) 
		VALUES (?, ?)
	`)
	if err != nil {
		fmt.Println("Error preparing statement:", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	for len(queue) > 0 && currentDepth < maxDepth {
		// Check database size before processing new depth
		exceeded, err := checkDatabaseSize(db, maxSizeBytes)
		if err != nil {
			fmt.Printf("Error checking database size: %v\n", err)
			dbCancel() // Cancel context on error
			return
		} else if exceeded {
			fmt.Println("Database size limit reached, stopping simulation")
			dbCancel() // Cancel context when DB is full
			return
		}

		select {
		case <-dbCtx.Done(): // Use dbCtx instead of ctx
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
				go func(workerID int) {
					name := fmt.Sprintf("Worker-%d", workerID)
					logGoroutine(name)
					defer logGoroutineExit(name)
					defer wg.Done()

					// Each worker gets its own connection to avoid WAL mode contention
					workerDB, err := sql.Open("sqlite3", "./chess.db")
					if err != nil {
						fmt.Printf("Worker %d failed to open DB: %v\n", workerID, err)
						return
					}
					defer workerDB.Close()

					for work := range workChan {
						// Check DB size at start of each work batch
						exceeded, err := checkDatabaseSize(workerDB, maxSizeBytes)
						if err != nil {
							fmt.Printf("Worker %d error checking DB size: %v\n", workerID, err)
							dbCancel()
							return
						}
						if exceeded {
							fmt.Printf("Worker %d detected DB size limit reached\n", workerID)
							dbCancel()
							return
						}

						select {
						case <-dbCtx.Done():
							return
						default:
							var localResults []*BoardRouteNode
							var boardsToStore []*components.ChessBoard
							var parentNodes []*BoardRouteNode // Track parent nodes for each board
							batchSize := 100                  // Adjust this value as needed

							for j := work.start; j < work.end; j++ {
								// Check DB size periodically during processing
								if j%10 == 0 { // Check every 10 states
									exceeded, err := checkDatabaseSize(workerDB, maxSizeBytes)
									if err != nil {
										fmt.Printf("Worker %d error checking DB size: %v\n", workerID, err)
										dbCancel()
										return
									}
									if exceeded {
										fmt.Printf("Worker %d detected DB size limit reached\n", workerID)
										dbCancel()
										return
									}
								}

								node := queue[j]
								currentBoard := getBoardStateByID(workerDB, node.StateID)
								if currentBoard == nil {
									errChan <- fmt.Errorf("nil board state for ID %d", node.StateID)
									continue
								}

								for y, row := range currentBoard.Board {
									for x, piece := range row {
										if piece != nil && piece.GetColor() == currentBoard.NextTurn {
											possibleMoves := piece.GetPossibleMoves(*currentBoard, components.Coordinates{X: x, Y: y}, false)
											for _, newBoard := range possibleMoves {
												newBoard.NextTurn = !currentBoard.NextTurn
												boardsToStore = append(boardsToStore, newBoard)
												parentNodes = append(parentNodes, node)

												// Process batch if we've reached the batch size
												if len(boardsToStore) >= batchSize {
													ids, err := storeBoardStatesBatch(workerDB, boardsToStore)
													if err != nil {
														errChan <- fmt.Errorf("failed to store board states batch: %v", err)
														continue
													}

													// Create nodes and store relations for the batch
													for i, id := range ids {
														newNode := &BoardRouteNode{
															StateID:      id,
															NextStateIDs: []int{},
														}
														parentNodes[i].NextStateIDs = append(parentNodes[i].NextStateIDs, id)
														if err := storeNodeRelation(workerDB, parentNodes[i].StateID, id); err != nil {
															errChan <- fmt.Errorf("failed to store node relation: %v", err)
															continue
														}
														localResults = append(localResults, newNode)
													}

													// Clear the batches
													boardsToStore = boardsToStore[:0]
													parentNodes = parentNodes[:0]
												}
											}
										}
									}
								}

								// Add size check inside worker
								exceeded, err := checkDatabaseSize(workerDB, maxSizeBytes)
								if err != nil || exceeded {
									dbCancel() // Cancel context if DB is full
									return
								}
							}

							// Process any remaining boards in the final batch
							if len(boardsToStore) > 0 {
								ids, err := storeBoardStatesBatch(workerDB, boardsToStore)
								if err != nil {
									errChan <- fmt.Errorf("failed to store final board states batch: %v", err)
								} else {
									for i, id := range ids {
										newNode := &BoardRouteNode{
											StateID:      id,
											NextStateIDs: []int{},
										}
										parentNodes[i].NextStateIDs = append(parentNodes[i].NextStateIDs, id)
										if err := storeNodeRelation(workerDB, parentNodes[i].StateID, id); err != nil {
											errChan <- fmt.Errorf("failed to store node relation: %v", err)
											continue
										}
										localResults = append(localResults, newNode)
									}
								}
							}

							select {
							case <-dbCtx.Done():
								return
							case results <- localResults:
							}
						}
					}
				}(i)
			}

			// Distribute work
			for i := 0; i < levelSize; i += batchSize {
				end := i + batchSize
				if end > levelSize {
					end = levelSize
				}

				select {
				case <-dbCtx.Done(): // Use dbCtx instead of ctx
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
				// Check database size before processing stored states
				exceeded, err := checkDatabaseSize(db, maxSizeBytes)
				if err != nil {
					fmt.Printf("Error checking database size: %v\n", err)
					dbCancel() // Cancel context on error
					return
				} else if exceeded {
					fmt.Println("Database size limit reached while processing stored states, stopping simulation")
					dbCancel() // Cancel context when DB is full
					return
				}

				// Count remaining states at current depth
				var remainingCount int
				err = db.QueryRow("SELECT COUNT(*) FROM excess_queue WHERE depth = ?", currentDepth).Scan(&remainingCount)
				if err != nil {
					fmt.Println("Error counting remaining states:", err)
					break
				}

				if remainingCount == 0 {
					break
				}

				fmt.Printf("Processing %d remaining states at depth %d\n", remainingCount, currentDepth)

				// Process stored states in smaller batches
				batchSize := 1000
				for i := 0; i < remainingCount; i += batchSize {
					select {
					case <-dbCtx.Done():
						fmt.Println("Processing cancelled, stopping excess queue processing...")
						return
					default:
						// Check size before each batch
						exceeded, err := checkDatabaseSize(db, maxSizeBytes)
						if err != nil {
							fmt.Printf("Error checking database size: %v\n", err)
							dbCancel()
							return
						} else if exceeded {
							fmt.Println("Database size limit reached during batch processing, stopping simulation")
							dbCancel()
							return
						}

						// Fetch next batch of stored states
						rows, err := db.Query("SELECT state_id FROM excess_queue WHERE depth = ? LIMIT ? OFFSET ?",
							currentDepth, batchSize, i)
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
												if err := storeNodeRelation(db, node.StateID, newStateID); err != nil {
													fmt.Printf("Error storing node relation: %v\n", err)
													continue
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
				}
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

func storeBoardStatesBatch(db *sql.DB, boards []*components.ChessBoard) ([]int, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback() // Will be ignored if transaction is committed

	// Prepare the insert statement
	stmt, err := tx.Prepare("INSERT INTO board_states (state) VALUES (?)")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	var ids []int
	for _, board := range boards {
		data, err := json.Marshal(board)
		if err != nil {
			continue
		}

		// Try to find existing state first
		var existingID int
		err = tx.QueryRow("SELECT id FROM board_states WHERE state = ?", string(data)).Scan(&existingID)
		if err == nil {
			// State already exists
			ids = append(ids, existingID)
			continue
		}

		// State doesn't exist, insert it
		result, err := stmt.Exec(string(data))
		if err != nil {
			continue
		}

		id, err := result.LastInsertId()
		if err != nil {
			continue
		}

		ids = append(ids, int(id))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return ids, nil
}

func getDiskSpace() (uint64, uint64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return 0, 0, err
	}

	// Available bytes = blocks * size
	available := stat.Bavail * uint64(stat.Bsize)
	total := stat.Blocks * uint64(stat.Bsize)

	return available, total, nil
}

// Add this function to get current goroutine count
func getGoroutineCount() int {
	return runtime.NumGoroutine()
}

// Modify the logging functions
func logGoroutine(name string) {
	fmt.Printf("\n\033[1;32m=== [%s] GOROUTINE START === %s (Total: %d) ===\033[0m\n",
		time.Now().Format("15:04:05"),
		name,
		getGoroutineCount())
}

func logGoroutineExit(name string) {
	fmt.Printf("\n\033[1;31m=== [%s] GOROUTINE END === %s (Total: %d) ===\033[0m\n",
		time.Now().Format("15:04:05"),
		name,
		getGoroutineCount())
}

func main() {
	// Get and display available disk space
	available, total, err := getDiskSpace()
	if err != nil {
		fmt.Printf("Error getting disk space: %v\n", err)
	} else {
		fmt.Printf("Disk space:\n")
		fmt.Printf("  Total: %.2f GB\n", float64(total)/1024/1024/1024)
		fmt.Printf("  Available: %.2f GB\n", float64(available)/1024/1024/1024)
	}

	// Get max database size from user
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter maximum database size in GB (e.g., 10): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	maxSizeGB, err := strconv.ParseFloat(input, 64)
	if err != nil {
		fmt.Println("Invalid input, using default of 10GB")
		maxSizeGB = 10
	}

	// Convert GB to bytes for SQLite
	maxPages := int64(maxSizeGB * 1024 * 1024 * 1024 / 4096) // 4KB per page

	// Start goroutine monitoring
	go func() {
		logGoroutine("Goroutine-Monitor")
		defer logGoroutineExit("Goroutine-Monitor")
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			fmt.Printf("\n\033[1;33m=== [%s] GOROUTINE COUNT === Total: %d ===\033[0m\n",
				time.Now().Format("15:04:05"),
				getGoroutineCount())
		}
	}()

	db, err := sql.Open("sqlite3", "./chess.db")
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}
	defer db.Close()

	// Enable WAL mode and optimize SQLite settings with size limit
	_, err = db.Exec(fmt.Sprintf(`
		PRAGMA journal_mode=WAL;
		PRAGMA synchronous=NORMAL;
		PRAGMA cache_size=10000;
		PRAGMA temp_store=MEMORY;
		PRAGMA mmap_size=30000000000;
		PRAGMA max_page_count=%d;
	`, maxPages))
	if err != nil {
		fmt.Println("Error setting SQLite pragmas:", err)
		return
	}

	// Add function to periodically check database size
	go func() {
		logGoroutine("DB-Size-Monitor")
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			var size int64
			err := db.QueryRow("SELECT page_count * page_size FROM pragma_page_count, pragma_page_size").Scan(&size)
			if err != nil {
				fmt.Printf("Error getting DB size: %v\n", err)
				continue
			}
			fmt.Printf("Current database size: %.2f GB\n", float64(size)/1024/1024/1024)
		}
	}()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS board_states (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			state TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS node_relations (
			parent_id INTEGER,
			child_id INTEGER,
			FOREIGN KEY(parent_id) REFERENCES board_states(id),
			FOREIGN KEY(child_id) REFERENCES board_states(id),
			PRIMARY KEY(parent_id, child_id)
		);
		CREATE INDEX IF NOT EXISTS idx_node_relations_parent 
		ON node_relations(parent_id);
	`)
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
		logGoroutine("Signal-Handler")
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

	maxSizeBytes := int64(maxSizeGB * 1024 * 1024 * 1024)

	// Run simulation in a separate goroutine
	go func() {
		logGoroutine("Main-Simulation")
		simulateGamesWithSQLite(ctx, rootNode, db, maxDepth, maxSizeBytes)
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

func storeNodeRelation(db *sql.DB, parentID, childID int) error {
	// Use INSERT OR IGNORE to handle potential duplicates
	_, err := db.Exec(`
		INSERT OR IGNORE INTO node_relations (parent_id, child_id) 
		VALUES (?, ?)`,
		parentID, childID)
	if err != nil {
		return fmt.Errorf("failed to store node relation: %v", err)
	}
	return nil
}

func getChildNodes(db *sql.DB, parentID int) ([]int, error) {
	rows, err := db.Query(`
		SELECT child_id 
		FROM node_relations 
		WHERE parent_id = ?
		ORDER BY child_id
	`, parentID)
	if err != nil {
		return nil, fmt.Errorf("error querying child nodes: %v", err)
	}
	defer rows.Close()

	var childIDs []int
	for rows.Next() {
		var childID int
		if err := rows.Scan(&childID); err != nil {
			return nil, fmt.Errorf("error scanning child ID: %v", err)
		}
		childIDs = append(childIDs, childID)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %v", err)
	}

	return childIDs, nil
}

func traverseTree(node *BoardRouteNode, db *sql.DB) {
	reader := bufio.NewReader(os.Stdin)
	currentNodeID := node.StateID
	var history []int

	for {
		fmt.Println("\nCurrent Board State (ID:", currentNodeID, "):")
		board := getBoardStateByID(db, currentNodeID)
		if board == nil {
			fmt.Println("Error: Could not retrieve board state")
			break
		}
		board.ToString()

		// Get child nodes from database
		childIDs, err := getChildNodes(db, currentNodeID)
		if err != nil {
			fmt.Printf("Error getting child nodes: %v\n", err)
			break
		}

		fmt.Println("\nAvailable Moves:")
		if len(childIDs) == 0 {
			fmt.Println("No further moves available.")
			if len(history) > 0 {
				fmt.Println("Press 'b' to go back or any other key to exit")
			} else {
				break
			}
		} else {
			for i, childID := range childIDs {
				childBoard := getBoardStateByID(db, childID)
				if childBoard != nil {
					fmt.Printf("%d: Move to state ID %d (%s to move)\n",
						i, childID,
						map[bool]string{true: "White", false: "Black"}[childBoard.NextTurn])
				}
			}
		}

		fmt.Print("\nEnter move number, 'b' to go back, or 'q' to quit: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "b":
			if len(history) > 0 {
				currentNodeID = history[len(history)-1]
				history = history[:len(history)-1]
			} else {
				fmt.Println("Cannot go back further")
			}
		case "q":
			return
		default:
			moveIndex, err := strconv.Atoi(input)
			if err != nil || moveIndex < 0 || moveIndex >= len(childIDs) {
				fmt.Println("Invalid input. Please enter a valid move number.")
				continue
			}
			history = append(history, currentNodeID)
			currentNodeID = childIDs[moveIndex]
		}
	}
}

// Add a helper function to debug the node relations
func debugNodeRelations(db *sql.DB, nodeID int) {
	rows, err := db.Query(`
		SELECT parent_id, child_id 
		FROM node_relations 
		WHERE parent_id = ? OR child_id = ?
	`, nodeID, nodeID)
	if err != nil {
		fmt.Printf("Error querying relations: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Printf("\nRelations for node %d:\n", nodeID)
	for rows.Next() {
		var parentID, childID int
		if err := rows.Scan(&parentID, &childID); err != nil {
			fmt.Printf("Error scanning row: %v\n", err)
			continue
		}
		fmt.Printf("Parent: %d -> Child: %d\n", parentID, childID)
	}
}
