package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"
)

// MAXIMIZER, MINIMIZER, UNSET
// are used to denote which player, and also
// as indexes into arrays for too-clever output and
// win/loss indicators.
const (
	MAXIMIZER = 1  // Computer plays MAXIMIZER
	MINIMIZER = -1 // Computer has human play MINIMIZER
	UNSET     = 0
	WIN       = 10000
	LOSS      = -10000
)

// Board - internal representation of a traditional Kalah board
type Board struct {
	maxpits [7]int
	minpits [7]int
	player  int // which player made the move resulting in this configuration
}

type chooserFunction func(bd Board, print bool) (bestpit int, bestvalue int)

type player struct {
	name   string
	bd     Board
	moveFn chooserFunction
}

// MCTS holds values that func chooseMonteCarlo() needs, but
// aren't passed in as arguments.
type MCTS struct {
	iterations int
	uctk       float64
}

type AlphaBeta struct {
	maxPly int
}

var winningStonesCount int

func main() {

	player1Type := flag.String("1", "M", "first player type")
	player2Type := flag.String("2", "A", "second player type")
	maxDepthPtr := flag.Int("d", 6, "maximum lookahead depth, moves for each side")
	stoneCountPtr := flag.Int("n", 4, "number of stones per pit")
	iterationPtr := flag.Int("i", 200000, "Number of iterations for MCTS")
	uctkPtr := flag.Float64("U", 1.414, "UCTK factor, MCTS only")
	flag.Parse()

	winningStonesCount = 6 * *stoneCountPtr

	maximizer := constructPlayer(*player1Type, *stoneCountPtr, *maxDepthPtr, *iterationPtr, *uctkPtr)
	minimizer := constructPlayer(*player2Type, *stoneCountPtr, *maxDepthPtr, *iterationPtr, *uctkPtr)

	rand.Seed(time.Now().UTC().UnixNano())

	// func main's copy of the board.
	var bd Board

	for i := 0; i < 6; i++ {
		bd.maxpits[i] = *stoneCountPtr
		bd.minpits[i] = *stoneCountPtr
	}

	player := MAXIMIZER

GAMELOOP:
	for {
		fmt.Printf("%v\n> ", bd)
		_, err := fmt.Scanf("\n")
		if err != nil {
			log.Print(err)
		}

		var pit, value int
		var minNxt, maxNxt int

		switch player {
		case MAXIMIZER:
			pit, value = maximizer.moveFn(maximizer.bd, false)
			fmt.Printf("%s chooses %d (%d)\n", maximizer.name, pit, value)
			maxNxt, _ = makeMove(&(maximizer.bd), pit, MAXIMIZER)
			minNxt, _ = makeMove(&(minimizer.bd), pit, MINIMIZER)
			if maxNxt != (0 - minNxt) {
				fmt.Printf("maximizer says %d goes next\n", maxNxt)
				fmt.Printf("minimizer says %d goes next\n", 0-minNxt)
			}
		case MINIMIZER:
			pit, value = minimizer.moveFn(minimizer.bd, false)
			fmt.Printf("%s chooses %d (%d)\n", minimizer.name, pit, value)
			minNxt, _ = makeMove(&(minimizer.bd), pit, MAXIMIZER)
			maxNxt, _ = makeMove(&(maximizer.bd), pit, MINIMIZER)
			if maxNxt != (0 - minNxt) {
				fmt.Printf("maximizer says %d goes next\n", maxNxt)
				fmt.Printf("minimizer says %d goes next\n", 0-minNxt)
			}
		}

		player, _ = makeMove(&bd, pit, player)
		gameEnd, winner := checkEnd(&bd)
		compare3(&bd, &(maximizer.bd), &(minimizer.bd))
		if player != maxNxt || player != (0-minNxt) {
			fmt.Printf("referee   says %d goes next\n", player)
			fmt.Printf("maximizer says %d goes next\n", maxNxt)
			fmt.Printf("minimizer says %d goes next\n", 0-minNxt)
		}
		if gameEnd {
			w := "player 1"
			if winner == MINIMIZER {
				w = "player 2"
			}
			fmt.Printf("Game over, %s won\n", w)
			break GAMELOOP
		}
	}
	fmt.Printf("Final:\n%v\n", bd)
}

func (p Board) String() string {
	var top, mid, bot string

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

	return top + mid + bot
}

func (ab *AlphaBeta) chooseMove(bd Board, print bool) (bestpit int, bestvalue int) {
	return chooseAlphaBeta(bd, ab.maxPly, print)
}

func chooseAlphaBeta(bd Board, maxPly int, print bool) (bestpit int, bestvalue int) {
	bestvalue = 2 * LOSS // -infinity
	bestpit = 0
	var bd2 Board
	for pit, stones := range bd.maxpits[0:6] {
		if stones > 0 {
			copy(bd2.maxpits[:], bd.maxpits[:])
			copy(bd2.minpits[:], bd.minpits[:])
			bd2.player = bd.player

			makeMove(&bd2, pit, MAXIMIZER)
			var value int
			if end, winner := checkEnd(&bd2); end {
				switch winner {
				case MAXIMIZER:
					value = WIN
				case MINIMIZER:
					value = LOSS
				default: // end of game, but no winner
					value = 0
				}
			} else {
				value = alphaBeta(&bd2, 1, MINIMIZER, 2*LOSS, 2*WIN, maxPly)
			}
			if value > bestvalue {
				bestvalue = value
				bestpit = pit
			}
			// makeMove() does a lot to bd2, just dump it.
		}
	}
	return bestpit, bestvalue
}

// alphaBeta does alpha-beta minimaxing. Computer is maximizer, human is minimizer.
// Pass current game board (bd *Board) by reference to avoid having the compiler
// create struct-copying code for each call to alphaBeta.
func alphaBeta(bd *Board, ply, player, alpha, beta int, maxPly int) (value int) {
	if ply > maxPly {
		// static value function: difference between pots less ply depth,
		// so that all things equal, choose the shortest path to a win,
		// plus some empirical amount of the seeds in computer's pits.
		return (bd.maxpits[6] - bd.minpits[6]) - ply +
			(bd.maxpits[0]+bd.maxpits[1]+bd.maxpits[2]+bd.maxpits[3]+bd.maxpits[4]+2*bd.maxpits[5])/3
	}
	// checkEnd() should get the case where someone already has
	// more than half the stones in their pot, so alphaBeta()
	// only has to do depth check

	switch player {
	case MAXIMIZER:
		var bd2 Board
		for pit, stones := range bd.maxpits[0:6] {
			if stones != UNSET {
				copy(bd2.maxpits[:], bd.maxpits[:])
				copy(bd2.minpits[:], bd.minpits[:])
				bd2.player = bd.player
				nextplayer, plydelta := makeMove(&bd2, pit, player)
				if end, winner := checkEnd(&bd2); end {
					switch winner {
					case MAXIMIZER:
						value = WIN - ply
					case MINIMIZER:
						value = LOSS + ply
					default:
						value = 0
					}
				} else {
					value = alphaBeta(&bd2, ply+plydelta, nextplayer, alpha, beta, maxPly)
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
		var bd2 Board
		for pit, stones := range bd.minpits[0:6] {
			if stones != 0 {
				copy(bd2.maxpits[:], bd.maxpits[:])
				copy(bd2.minpits[:], bd.minpits[:])
				bd2.player = bd.player
				nextplayer, plydelta := makeMove(&bd2, pit, player)
				if end, winner := checkEnd(&bd2); end {
					switch winner {
					case MAXIMIZER:
						value = WIN - ply
					case MINIMIZER:
						value = LOSS + ply
					default:
						value = 0
					}
				} else {
					value = alphaBeta(&bd2, ply+plydelta, nextplayer, alpha, beta, maxPly)
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

func makeMove(bd *Board, pit int, player int) (nextplayer int, plydelta int) {
	var sides [2]*[7]int

	if pit > 5 {
		fmt.Printf("problem player %d move %d, pit > 6: %s\n", player, pit, bd)
	}

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

	if hand == 0 {
		panic(fmt.Errorf("problem player %d move %d, empty pit:\n%s\n", player, pit, bd))
	}

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
	bd.player = player
	if bonusmove {
		nextplayer = player
		plydelta = 0
	}
	return nextplayer, plydelta
}

// checkEnd figures out if the current game board, passed by reference
// to avoid compiler-generated struct copying, represents a win/loss/tie
// and for which player.
func checkEnd(bd *Board) (end bool, winner int) {
	if bd.maxpits[6] > winningStonesCount {
		return true, MAXIMIZER
	}
	if bd.minpits[6] > winningStonesCount {
		return true, MINIMIZER
	}
	winner = UNSET
	maxsidesum := 0
	minsidesum := 0
	for i := 0; i < 6; i++ {
		maxsidesum += bd.maxpits[i]
		minsidesum += bd.minpits[i]
	}
	if minsidesum == 0 || maxsidesum == 0 {
		end = true
		for i := 0; i < 6; i++ {
			bd.maxpits[i] = UNSET
			bd.minpits[i] = UNSET
		}
		bd.maxpits[6] += maxsidesum
		bd.minpits[6] += minsidesum
	}
	if end {
		winner = bd.maxpits[6] - bd.minpits[6]
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

type gameState struct {
	board Board
}

type Node struct {
	move         int
	player       int
	childNodes   []*Node
	untriedMoves []int
	parent       *Node
	visits       int
	wins         float64
}

// chooseMonteCarlo - based on current board, return the best pit
// for MAXIMIZER to pick up and drop down the board.
func (p *MCTS) chooseMonteCarlo(bd Board, print bool) (bestpit int, value int) {

	root := &Node{
		player:       MINIMIZER, // opponent made last move
		untriedMoves: make([]int, 0, 6),
	}
	// by definition the next player is MAXIMIZER.
	// Fill in MAXIMIMIZER's untried moves
	for i := 0; i < 6; i++ {
		if bd.maxpits[i] != 0 {
			root.untriedMoves = append(root.untriedMoves, i)
		}
	}

	state := &Board{}

	for iter := 0; iter < p.iterations; iter++ {
		// reset game state tracker
		for i := 0; i < 7; i++ {
			state.maxpits[i] = bd.maxpits[i]
			state.minpits[i] = bd.minpits[i]
		}
		state.player = root.player
		nextPlayer := -root.player

		node := root

		// Selection
		for len(node.untriedMoves) == 0 && len(node.childNodes) > 0 {
			node = node.selectBestChild(p.uctk)
			// Filling state from a game tree, so use node.move, node.player,
			// ignoring nextPlayer for now.
			nextPlayer, _ = makeMove(state, node.move, node.player)
		}

		gameEnd, winner := checkEnd(state)

		// Expansion
		if !gameEnd && len(node.untriedMoves) > 0 {
			mv := node.randomUntried()

			nextPlayer, _ = makeMove(state, mv, nextPlayer)
			node = node.addChild(mv, nextPlayer, state)

			gameEnd, winner = checkEnd(state)
		}

		// Simulation
		if !gameEnd {
			for !gameEnd {
				mv := state.randomMove(nextPlayer)
				nextPlayer, _ = makeMove(state, mv, nextPlayer)
				gameEnd, winner = checkEnd(state)
			}
		}

		// Back propagation
		for node != nil {
			node.visits++
			if winner == node.player {
				node.wins++
			} else if winner == 0 {
				node.wins += 0.5
			}
			node = node.parent
		}
	}

	// Select child move with the largest number of visits
	bestChild := root.childNodes[0]
	mostVisits := bestChild.visits

	for _, c := range root.childNodes[1:] {
		if c.visits > mostVisits {
			bestChild = c
			mostVisits = bestChild.visits
		}
	}

	return bestChild.move, int(bestChild.wins / float64(bestChild.visits) * 100.)
}

func (bd *Board) randomMove(player int) int {
	if player == MAXIMIZER {
		for {
			i := rand.Intn(6)
			if bd.maxpits[i] != 0 {
				return i
			}
		}
	}
	for {
		i := rand.Intn(6)
		if bd.minpits[i] != 0 {
			return i
		}
	}
	fmt.Printf("Board.randomMove(%d) shouldn't get here\n", player)
	return 0
}

func (n *Node) randomUntried() int {
	ln := len(n.untriedMoves)
	randIdx := rand.Intn(ln)
	ln--
	mv := n.untriedMoves[randIdx]
	n.untriedMoves[randIdx] = n.untriedMoves[ln]
	n.untriedMoves = n.untriedMoves[:ln]
	return mv
}

func (n *Node) addChild(mv int, nextPlayer int, state *Board) *Node {
	if mv > 5 {
		fmt.Printf("addChild, move %d illegal\n", mv)
		fmt.Printf("parent node: %d/%d, untried moves %v\n",
			n.move, n.player, n.untriedMoves)
		fmt.Printf("next player %d, state.player %d\n%s\n",
			nextPlayer, state.player, state)
		panic("bad child move")
	}
	newChild := &Node{
		move:         mv,
		player:       state.player,
		parent:       n,
		untriedMoves: remainingMoves(state, nextPlayer),
	}
	n.childNodes = append(n.childNodes, newChild)
	return newChild
}

func (n *Node) selectBestChild(uctk float64) *Node {
	bestScore := n.childNodes[0].ucb1(uctk)
	bestChild := n.childNodes[0]
	for _, c := range n.childNodes[1:] {
		score := c.ucb1(uctk)
		if score > bestScore {
			bestScore = score
			bestChild = c
		}
	}
	return bestChild
}

func remainingMoves(bd *Board, player int) []int {
	mvs := make([]int, 0, 6)
	if player == MAXIMIZER {
		for i := 0; i < 6; i++ {
			if bd.maxpits[i] != 0 {
				mvs = append(mvs, i)
			}
		}
		return mvs
	}
	for i := 0; i < 6; i++ {
		if bd.minpits[i] != 0 {
			mvs = append(mvs, i)
		}
	}
	return mvs
}

func (n *Node) ucb1(uctk float64) float64 {
	v := float64(n.visits)
	return n.wins/v +
		uctk*math.Sqrt(math.Log(float64(n.parent.visits+1))/v)
}

/*
type chooserFunction func(bd Board, print bool) (bestpit int, bestvalue int)
type player struct {
	bd     Board
	moveFn chooserFunction
}
*/

func constructPlayer(typ string, stonesPerPit int, maxDepth int, mctsIterations int, uctk float64) *player {
	var p player

	for i := 0; i < 6; i++ {
		p.bd.maxpits[i] = stonesPerPit
		p.bd.minpits[i] = stonesPerPit
	}

	switch typ {
	case "M": // MCTS+UCB1
		mcts := &MCTS{iterations: mctsIterations, uctk: uctk}
		p.moveFn = mcts.chooseMonteCarlo
		p.name = "MCTS"
	case "A": // Alpha-beta minimaxing
		// func alphaBeta(bd *Board, ply, player, alpha, beta int, maxPly int) (value int) {
		ab := &AlphaBeta{maxPly: 2 * maxDepth}
		p.moveFn = ab.chooseMove
		p.name = "A/B"
	default:
		fmt.Fprintf(os.Stderr, "Unknown player type %q\n", typ)
	}
	return &p
}

func compare3(ref, max, min *Board) {
	for i := 0; i < 7; i++ {
		if ref.maxpits[i] != max.maxpits[i] ||
			ref.maxpits[i] != min.minpits[i] ||
			ref.minpits[i] != max.minpits[i] ||
			ref.minpits[i] != min.maxpits[i] {
			fmt.Printf("Boards disagree:\n")
			fmt.Printf("referee:\n%v\n", ref)
			fmt.Printf("maximizer:\n%v\n", max)
			fmt.Printf("minimizer:\n%v\n", min)
		}
	}
}
