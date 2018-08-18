package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	MAXIMIZER = 1
	MINIMIZER = -1
	UNSET     = 0
	WIN       = 10000
	LOSS      = -10000
)

type Board struct {
	maxpits [7]int
	minpits [7]int
	reverse bool
}

var maxPly int = 16

func main() {

	computerFirstPtr := flag.Bool("C", false, "Computer takes first move")
	maxDepthPtr := flag.Int("d", 8, "maximum lookahead depth, moves for each side")
	stoneCountPtr := flag.Int("n", 4, "number of stones per pit")
	reversePtr := flag.Bool("R", false, "Reverse printed board, top-to-bottom")
	flag.Parse()

	var bd Board
	if *reversePtr {
		bd.reverse = true
	}

	for i := 0; i < 6; i++ {
		bd.maxpits[i] = *stoneCountPtr
		bd.minpits[i] = *stoneCountPtr
	}

	player := MINIMIZER
	if *computerFirstPtr {
		player = MAXIMIZER
	}

	maxPly = 2 * *maxDepthPtr

	for {
		var pit, value int
		fmt.Printf("%v\n", bd)
		switch player {
		case MINIMIZER:
			pit = readMove(bd, true)
		case MAXIMIZER:
			before := time.Now()
			pit, value = chooseMove(bd, true)
			et := time.Now().Sub(before)
			fmt.Printf("Computer chooses %d (%d) [%v]\n---\n", pit, value, et)
		}
		player, _ = makeMove(&bd, pit, player)
		gameEnd, winner := checkEnd(&bd)
		if gameEnd {
			w := "computer"
			if winner == MINIMIZER {
				w = "human"
			}
			fmt.Printf("Game over, %s won\n", w)
			break
		}
	}
	fmt.Printf("Final:\n%v\n", bd)
}

func (p Board) String() string {
	var top, mid, bot string

	if p.reverse {
		top = fmt.Sprintf("   %2d %2d %2d %2d %2d %2d\n",
			p.minpits[5],
			p.minpits[4],
			p.minpits[3],
			p.minpits[2],
			p.minpits[1],
			p.minpits[0])
		bot = fmt.Sprintf("   %2d %2d %2d %2d %2d %2d",
			p.maxpits[0],
			p.maxpits[1],
			p.maxpits[2],
			p.maxpits[3],
			p.maxpits[4],
			p.maxpits[5])
		mid = fmt.Sprintf("%2d                   %2d\n", p.minpits[6], p.maxpits[6])
	} else {

		top = fmt.Sprintf("   %2d %2d %2d %2d %2d %2d\n",
			p.maxpits[5],
			p.maxpits[4],
			p.maxpits[3],
			p.maxpits[2],
			p.maxpits[1],
			p.maxpits[0])
		bot = fmt.Sprintf("   %2d %2d %2d %2d %2d %2d",
			p.minpits[0],
			p.minpits[1],
			p.minpits[2],
			p.minpits[3],
			p.minpits[4],
			p.minpits[5])
		mid = fmt.Sprintf("%2d                   %2d\n", p.maxpits[6], p.minpits[6])
	}

	return top + mid + bot
}

func chooseMove(bd Board, print bool) (bestpit int, bestvalue int) {
	bestvalue = 2 * LOSS
	bestpit = 0
	for pit, stones := range bd.maxpits[0:6] {
		if stones > 0 {
			bd2 := bd
			makeMove(&bd2, pit, MAXIMIZER)
			end, winner := checkEnd(&bd2)
			var value int
			if !end {
				value = alphaBeta(bd2, 1, MINIMIZER, 2*LOSS, 2*WIN)
			} else {
				switch winner {
				case MAXIMIZER:
					value = WIN
				case MINIMIZER:
					value = LOSS
				default: //
					value = bd2.maxpits[6] - bd2.minpits[6]
				}
			}
			// fmt.Printf("pit %d/%d, value %d, best value %d for %d\n", pit, stones, value, bestpit, bestvalue)
			if value > bestvalue {
				bestvalue = value
				bestpit = pit
			}
			// makeMove() does a lot to bd2, just dump it.
		}
	}
	return bestpit, bestvalue
}

func alphaBeta(bd Board, ply, player, alpha, beta int) (value int) {
	if ply > maxPly {
		return bd.maxpits[6] - bd.minpits[6] // low cost static value
	}
	switch player {
	case MAXIMIZER:
		value = 2 * LOSS
		for pit, stones := range bd.maxpits[0:6] {
			if stones != UNSET {
				bd2 := bd
				nextplayer, plydelta := makeMove(&bd2, pit, player)
				end, winner := checkEnd(&bd2)
				var n int
				if !end {
					n = alphaBeta(bd2, ply+plydelta, nextplayer, alpha, beta)
				} else {
					switch winner {
					case MAXIMIZER:
						value = WIN - ply
					case MINIMIZER:
						value = LOSS + ply
					default:
						value = 0
					}
				}

				if n > value {
					value = n
				}
				if value > alpha {
					alpha = value
				}
				if beta <= alpha {
					return value
				}
			}
		}
	case MINIMIZER:
		value = 2 * WIN // You can score greater than WIN
		for pit, stones := range bd.minpits[0:6] {
			if stones != 0 {
				bd2 := bd
				nextplayer, plydelta := makeMove(&bd2, pit, player)
				end, winner := checkEnd(&bd2)
				var n int
				if !end {
					n = alphaBeta(bd2, ply+plydelta, nextplayer, alpha, beta)
				} else {
					switch winner {
					case MAXIMIZER:
						value = WIN - ply
					case MINIMIZER:
						value = LOSS + ply
					default:
						value = 0
					}
				}
				if n < value {
					value = n
				}
				if value < beta {
					beta = value
				}
				if beta <= alpha {
					return value
				}
			}
		}
	}
	return value
}

func readMove(bd Board, print bool) (pit int) {
READMOVE:
	for {
		if print {
			fmt.Printf("Your move: ")
		}
		_, err := fmt.Scanf("%d\n", &pit)
		if err == io.EOF {
			os.Exit(0)
		}
		if err != nil {
			fmt.Printf("Failed to read: %v\n", err)
			os.Exit(1)
		}
		switch {
		case pit < 0 || pit > 5:
			if print {
				fmt.Printf("Choose a number between 0 and 5, try again\n")
			}
		case bd.minpits[pit] != UNSET:
			break READMOVE
		}
	}
	return pit
}

func makeMove(bd *Board, pit int, player int) (nextplayer int, plydelta int) {
	var sides [2]*[7]int

	nextplayer = -player
	plydelta = 1

	switch player {
	case MAXIMIZER:
		sides[0] = &(bd.maxpits)
		sides[1] = &(bd.minpits)
	case MINIMIZER:
		sides[0] = &(bd.minpits)
		sides[1] = &(bd.maxpits)
	}

	S := 0 // side of player is always 0
	hand := sides[S][pit]
	sides[S][pit] = UNSET

	bonusmove := false

	for i := pit + 1; hand > 0; {
		// last stone, on player's side, last pit is empty,
		// and pit across has stones.
		if hand == 1 && S == 0 && i < 6 && sides[S][i] == 0 && sides[S^1][5-i] > 0 {
			sides[S][6] += sides[S^1][5-i] + 1
			sides[S^1][5-i] = 0
			sides[S][i]-- // so no special cases just below
		}
		if !(S == 1 && i == 6) {
			sides[S][i]++
			hand--
		}
		if i == 6 {
			i = 0
			S ^= 1 // flip to other side of board
			if hand == 0 {
				bonusmove = true
			}
		} else {
			i++
		}
	}
	if bonusmove {
		nextplayer = player
		plydelta = 0
	}
	return nextplayer, plydelta
}

func checkEnd(bd *Board) (end bool, winner int) {
	winner = UNSET
	sidesum := 0
	for i := 0; i < 6; i++ {
		sidesum += bd.maxpits[i]
	}
	if sidesum == 0 {
		end = true
		otherleft := 0
		for i := 0; i < 6; i++ {
			otherleft += bd.minpits[i]
			bd.minpits[i] = UNSET
		}
		bd.minpits[6] += otherleft
		winner = bd.maxpits[6] - bd.minpits[6]
	} else {
		sidesum = 0
		for i := 0; i < 6; i++ {
			sidesum += bd.minpits[i]
		}
		if sidesum == 0 {
			end = true
			otherleft := 0
			for i := 0; i < 6; i++ {
				otherleft += bd.maxpits[i]
				bd.maxpits[i] = UNSET
			}
			bd.maxpits[6] += otherleft
			winner = bd.maxpits[6] - bd.minpits[6]
		}
	}
	if end {
		// Ties can happen, winner == 0 in that case, which == UNSET
		switch {
		case winner > 0:
			winner = MAXIMIZER
		case winner < 0:
			winner = MINIMIZER
		}
	}
	return end, winner
}
