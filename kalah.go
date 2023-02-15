package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/pprof"
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
	reverse bool
	player  int // which player made the move resulting in this configuration
}

type chooserFunction func(bd Board, moves []int, print bool) (bestpit int, bestvalue int)

// MCTS holds values that func chooseMonteCarlo() needs, but
// aren't passed in as arguments. Also keeps part of the *Node
// tree from func UCT() until the computer's next move.
type MCTS struct {
	moveNode   *Node
	iterations int
	uctk       float64
}

var maxPly = 16
var winningStonesCount int

var verbose bool

func main() {

	computerFirstPtr := flag.Bool("C", false, "Computer takes first move")
	verbosePtr := flag.Bool("v", false, "verbose MCTS output")
	maxDepthPtr := flag.Int("d", 6, "maximum lookahead depth, moves for each side")
	stoneCountPtr := flag.Int("n", 4, "number of stones per pit")
	reversePtr := flag.Bool("R", false, "Reverse printed board, top-to-bottom")
	monteCarloPtr := flag.Bool("M", false, "MCTS instead of alpha/beta minimax")
	profilePtr := flag.Bool("P", false, "Do CPU profiling")
	iterationPtr := flag.Int("i", 500000, "Number of iterations for MCTS")
	uctkPtr := flag.Float64("U", 1.414, "UCTK factor, MCTS only")
	flag.Parse()

	if *profilePtr {
		os.Remove("kalah.prof")
		f, err := os.Create("kalah.prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
		defer f.Close()
	}

	if *verbosePtr {
		verbose = true
	}

	var bd Board
	if *reversePtr {
		bd.reverse = true
	}

	for i := 0; i < 6; i++ {
		bd.maxpits[i] = *stoneCountPtr
		bd.minpits[i] = *stoneCountPtr
	}
	winningStonesCount = 6 * *stoneCountPtr

	player := MINIMIZER
	if *computerFirstPtr {
		player = MAXIMIZER
	}

	rand.Seed(time.Now().UTC().UnixNano())

	var chooseMove chooserFunction = chooseAlphaBeta

	if *monteCarloPtr {
		mcts := &MCTS{iterations: *iterationPtr, uctk: *uctkPtr}
		chooseMove = mcts.chooseMonteCarlo
	}

	maxPly = 2 * *maxDepthPtr

	var consecutiveMoves []int

	for {
		var pit, value int
		fmt.Printf("%v\n", bd)
		switch player {
		case MINIMIZER:
			pit = readMove(bd, true)
		case MAXIMIZER:
			// fmt.Printf("Moves between last computer move and now: %v\n", consecutiveMoves)
			before := time.Now()
			pit, value = chooseMove(bd, consecutiveMoves, true)
			et := time.Since(before)
			fmt.Printf("Computer chooses %d (%d) [%v]\n---\n", pit, value, et)
			consecutiveMoves = make([]int, 0)
		}
		consecutiveMoves = append(consecutiveMoves, pit)
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

func chooseAlphaBeta(bd Board, moves []int, print bool) (bestpit int, bestvalue int) {
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
	if bd.maxpits[6] > winningStonesCount {
		fmt.Fprintf(os.Stderr, "x")
		return WIN - ply
	}
	if bd.minpits[6] > winningStonesCount {
		fmt.Fprintf(os.Stderr, "y")
		return LOSS - ply
	}
	if ply > maxPly {
		return bd.maxpits[6] - bd.minpits[6] // low cost static value
	}
	// checkEnd() should get the case where someone already has
	// more than half the stones in their pot, so alphaBeta()
	// only has to do depth check
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
func (p *MCTS) chooseMonteCarlo(bd Board, pastMoves []int, print bool) (bestpit int, value int) {

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
		if verbose {
			fmt.Printf("\n\nIteration %d\n", iter)
		}
		// reset game state tracker
		for i := 0; i < 7; i++ {
			state.maxpits[i] = bd.maxpits[i]
			state.minpits[i] = bd.minpits[i]
		}
		state.player = root.player
		nextPlayer := -root.player

		node := root

		if verbose {
			fmt.Printf("0 game, %d, next %d:\n%v\n", state.player, nextPlayer, state)
		}

		// Selection
		for len(node.untriedMoves) == 0 && len(node.childNodes) > 0 {
			oldmove, oldplayer := node.move, node.player
			node = node.selectBestChild()
			if verbose {
				fmt.Printf("Best child of %d by %d:%d by %d\n", oldmove, oldplayer, node.move, node.player)
			}
			// Filling state from a game tree, so use node.move, node.player,
			// ignoring nextPlayer for now.
			nextPlayer, _ = makeMove(state, node.move, node.player)
			if verbose {
				fmt.Printf("after %d/%d, %d, next %d:\n%s\n", node.move, node.player, state.player, nextPlayer, state)
			}
		}

		if verbose {
			fmt.Printf("1 game, %d, next %d:\n%v\n", state.player, nextPlayer, state)
		}
		gameEnd, winner := checkEnd(state)
		if verbose {
			fmt.Printf("Game end %v, winner %d\n", gameEnd, winner)
		}

		// Expansion
		if !gameEnd && len(node.untriedMoves) > 0 {
			if verbose {
				fmt.Printf("Expansion, player %d, next %d, untried moves %v\n", node.player, nextPlayer, node.untriedMoves)
			}
			mv := node.randomUntried()
			if verbose {
				fmt.Printf("Expansion, player %d, chose move %d, untried moves %v\n", node.player, mv, node.untriedMoves)
			}

			nextPlayer, _ = makeMove(state, mv, nextPlayer)
			node = node.addChild(mv, nextPlayer, state)
			if verbose {
				fmt.Printf("2 game, %d:\n%v\n", state.player, state)
			}

			gameEnd, winner = checkEnd(state)
		}

		// Simulation
		if !gameEnd {
			if verbose {
				fmt.Printf("Simulation begins, %d:\n%v\n", nextPlayer, state)
			}
			for !gameEnd {
				mv := state.randomMove(nextPlayer)
				nextPlayer, _ = makeMove(state, mv, nextPlayer)
				gameEnd, winner = checkEnd(state)
			}
			if verbose {
				fmt.Printf("Simulation ends, %d, winner %d:\n%v\n", nextPlayer, winner, state)
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
	if verbose {
		fmt.Printf("new child of %d/%d: %d/%d, untried %v\n",
			n.move, n.player,
			newChild.move, newChild.player,
			newChild.untriedMoves,
		)
	}
	n.childNodes = append(n.childNodes, newChild)
	return newChild
}

func (n *Node) selectBestChild() *Node {
	bestScore := n.childNodes[0].ucb1()
	bestChild := n.childNodes[0]
	for _, c := range n.childNodes[1:] {
		score := c.ucb1()
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

func (n *Node) ucb1() float64 {
	v := float64(n.visits)
	return n.wins/v +
		1.414*math.Sqrt(math.Log(float64(n.parent.visits+1))/v)
}
