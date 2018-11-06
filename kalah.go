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

const (
	MAXIMIZER = 1
	MINIMIZER = -1
	UNSET     = 0
	WIN       = 10000
	LOSS      = -10000
	UCTK      = 1.00
)

type Board struct {
	maxpits [7]int
	minpits [7]int
	reverse bool
}

type chooserFunction func(bd Board, print bool) (bestpit int, bestvalue int)

type GameState struct {
	player        int
	board         Board
	cachedResults [3]float64
}

type Node struct {
	move         int // Move that got to this condition
	player       int // player that made the move
	parentNode   *Node
	childNodes   []*Node
	wins         float64
	visits       float64
	untriedMoves []int
}

type MCTS struct {
	movesNode  *Node
	iterations int
}

var maxPly int = 16
var winningStonesCount int

func main() {

	computerFirstPtr := flag.Bool("C", false, "Computer takes first move")
	maxDepthPtr := flag.Int("d", 6, "maximum lookahead depth, moves for each side")
	stoneCountPtr := flag.Int("n", 4, "number of stones per pit")
	reversePtr := flag.Bool("R", false, "Reverse printed board, top-to-bottom")
	monteCarloPtr := flag.Bool("M", false, "MCTS instead of alpha/beta minimax")
	profilePtr := flag.Bool("P", false, "Do CPU profiling")
	iterationPtr := flag.Int("i", 15000, "Number of iterations for MCTS")
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

	var chooseMove chooserFunction

	if *monteCarloPtr {
		mcts := &MCTS{}
		mcts.iterations = *iterationPtr
		chooseMove = mcts.chooseMonteCarlo
		rand.Seed(time.Now().UTC().UnixNano())
	} else {
		chooseMove = chooseAlphaBeta
	}

	maxPly = 2 * *maxDepthPtr

	for {
		var pit, value int
		fmt.Printf("%v\n", bd)
		switch player {
		case MINIMIZER:
			//pit = readMove(bd, true)
			pit = 5
		case MAXIMIZER:
			before := time.Now()
			pit, value = chooseMove(bd, true)
			et := time.Since(before)
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

func chooseAlphaBeta(bd Board, print bool) (bestpit int, bestvalue int) {
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
	if bd.maxpits[6] > winningStonesCount {
		return true, MAXIMIZER
	}
	if bd.minpits[6] > winningStonesCount {
		return true, MINIMIZER
	}
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

// chooseMonteCarlo - based on current board, return the best pit
// for MAXIMIZER to pick up and drop down the board.
func (p *MCTS) chooseMonteCarlo(bd Board, print bool) (bestpit int, value int) {
	fmt.Println("enter chooseMonteCarlo")
	fmt.Printf("bd:\n%s\n", bd.String())
	bestpit, bestvalue := UCT(bd, p.iterations, 1.00)
	return bestpit, int(bestvalue)
}

// UCT - based on board and player (who makes this move),
// return the best move and its value
func UCT(bd Board, itermax int, UCTK float64) (int, float64) {

	rootState := GameState{player: MINIMIZER, board: bd}
	rootNode := Node{player: MINIMIZER}
	rootNode.untriedMoves, _ = rootState.GetMoves()
	fmt.Printf("Root state %v\n", rootState)
	fmt.Printf("Root Node  %v\n", rootNode)

	for i := 0; i < itermax; i++ {

		node := &rootNode  // node moves up & down tree
		state := rootState // need to leave rootstate alone

		for len(node.untriedMoves) == 0 && len(node.childNodes) > 0 {
			node = node.UCTSelectChild(UCTK) // updates node: now a child of rootNode
			state.DoMove(node.move)          // updates state.player and state.board
		}

		// This condition creates a child node from an untried move
		// (if any exist), makes the move in state, and makes node
		// the child node.
		if len(node.untriedMoves) > 0 {
			m := node.untriedMoves[rand.Intn(len(node.untriedMoves))]
			state.DoMove(m)                 // update state.player to who makes *next* move, state.board
			node = node.AddChild(m, &state) // updates node with the child, and player value of state.player
			// node now represents m, the previously-untried move.
		}

		moves, endOfGame := state.GetMoves()

		// starting with current state, pick a random
		// branch of the game tree, all the way to a win/loss.
		for !endOfGame {
			m := moves[rand.Intn(len(moves))]
			state.DoMove(m)
			moves, endOfGame = state.GetMoves()
		}

		// state.board now points to a board where a player
		// won and the other lost, and it's a "descendant"
		// of the board in node. node isn't necessarily at
		// the end of the game. Trace back up the tree,
		// updating each node's wins and visit count.

		state.resetCachedResults()
		for ; node != nil; node = node.parentNode {
			node.Update(state.GetResult(node.player))
		}
	}

	fmt.Printf("End of UCT, rootnode: %v\nChildred:\n", rootNode)
	for _, childNode := range rootNode.childNodes {
		fmt.Printf("\t%v\n", childNode)
	}
	bs, bm := rootNode.bestMove(UCTK)
	fmt.Printf("UCT returns: %v\n", bm)
	return bm.move, bs
}

func (p *Node) bestMove(UCTK float64) (bestscore float64, bestmove *Node) {
	bestscore = math.SmallestNonzeroFloat64
	for _, c := range p.childNodes {
		ucb1 := c.UCB1(p.visits, UCTK)
		if ucb1 > bestscore {
			bestscore = ucb1
			bestmove = c
		}
	}
	return bestscore, bestmove
}

func (p *Node) UCTSelectChild(UCTK float64) *Node {
	_, n := p.bestMove(UCTK)
	if n == nil {
		fmt.Printf("UCTSelectChild returns nil\n")
		fmt.Printf("Node: %v\n", p)
	}
	return n
}

func (p *Node) UCB1(parentVisits float64, UCTK float64) float64 {
	return p.wins/(p.visits+math.SmallestNonzeroFloat64) + UCTK*math.Sqrt(2.*math.Log(parentVisits)/(p.visits+math.SmallestNonzeroFloat64))
}

// AddChild creates a new *Node with the state of st
// argument, takes move out of p.untriedMoves, adds
// the new *Node to the array of child nodes, returns
// the new *Node, which is then a child of p.
func (p *Node) AddChild(move int, st *GameState) *Node {

	n := &Node{move: move, parentNode: p, player: st.player}
	n.untriedMoves, _ = st.GetMoves()

	for i, m := range p.untriedMoves {
		if m == move {
			p.untriedMoves = append(p.untriedMoves[:i], p.untriedMoves[i+1:]...)
			break
		}
	}
	p.childNodes = append(p.childNodes, n)
	return n
}

func (p *Node) Update(result float64) {
	p.visits++
	p.wins += result
}

func NewGameState() *GameState {
	return &GameState{player: MINIMIZER}
}

func (p *GameState) Clone() *GameState {
	return &GameState{player: p.player, board: p.board}
}

func (p *GameState) resetCachedResults() {
	p.cachedResults[0] = -1
	p.cachedResults[1] = -1
	p.cachedResults[2] = -1
}

func (p *GameState) DoMove(move int) {
	nextPlayer, _ := makeMove(&(p.board), move, p.player)
	p.player = nextPlayer
}

func (p *GameState) GetMoves() (moves []int, endOfGame bool) {
	// Get here, p.board does not represent a win or a loss.

	if p.board.maxpits[6] >= winningStonesCount ||
		p.board.minpits[6] >= winningStonesCount {
		endOfGame = true
	}
	var side, other [7]int
	switch -p.player { // The moves have to be for the other player
	case MAXIMIZER:
		side = p.board.maxpits
		other = p.board.minpits
	case MINIMIZER:
		side = p.board.minpits
		other = p.board.maxpits
	}
	sidesum, othersum := 0, 0
	for i := 0; i < 6; i++ {
		sidesum += side[i]
		othersum += other[i]
		if side[i] != 0 {
			moves = append(moves, i)
		}
	}

	if sidesum == 0 || othersum == 0 {
		endOfGame = true
	}

	return moves, endOfGame
}

func (p *GameState) GetResult(playerJustMoved int) float64 {
	cached := p.cachedResults[playerJustMoved+1]
	if cached >= 0.0 {
		return cached
	}

	if p.board.maxpits[6] > p.board.minpits[6] {
		p.cachedResults[MAXIMIZER+1] = 1.0
		p.cachedResults[MINIMIZER+1] = 0.0
	} else if p.board.minpits[6] > p.board.maxpits[6] {
		p.cachedResults[MAXIMIZER+1] = 0.0
		p.cachedResults[MINIMIZER+1] = 1.0
	}

	return p.cachedResults[playerJustMoved+1]
}

func (p *GameState) String() string {
	rep := "MAX\n"
	if p.player == MINIMIZER {
		rep = "MIN\n"
	}
	rep += p.board.String()
	return rep
}

func (p *Node) String() string {
	return fmt.Sprintf("%p %d/%d - %.0f:%.0f, %p",
		p, p.move, p.player, p.wins, p.visits, p.parentNode)
}
