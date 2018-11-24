# Kalah, an ancient game

Another implementation of [the game of Kalah](https://en.wikipedia.org/wiki/Kalah).
I implemented Wikipedia's rules exactly, as they were also what I was
accustomed to playing.

## Compiling

    $ git clone https://github.com/bediger4000/kalah.git $GOPATH/src/kalah
    $ go build kalah
    $ ./kalah

You don't have to install it anywhere - it runs in place.
It has no configuration file(s).

## Use

   Usage of ./kalah:
    -C    Computer takes first move
    -M    MCTS instead of alpha/beta minimax
    -P    Do CPU profiling
    -R    Reverse printed board, top-to-bottom
    -d int
          maximum lookahead depth, moves for each side (default 6)
    -i int
          Number of iterations for MCTS (default 500000)
    -n int
          number of stones per pit (default 4)

"MCTS" means [Monte Carlo Tree Search](http://mcts.ai/).
It defaults to deciding what move to make by using Alpha/Beta minimaxing.

## Design

I used the Wikipedia article on [Alpha/Beta minimaxing](https://en.wikipedia.org/wiki/Alpha%E2%80%93beta_pruning).
I should implemented one level of threading.
Static value calculated as difference of player's pots or stores.

The MCTS code follows the [MCTS example code](http://mcts.ai/code/python.html),
transliterated into Go for [squava](https://github.com/bediger4000/squava),
then mutated significantly to allow for Kalah's bonus moves.
It was suprisingly difficult to get bonus moves correct for MCTS.

### Bonus move

This variant has a bonus move.
Players get an extra move if they drop the last stone in their
hand into their own pot/store.
The Alpha/Beta minimaxing code takes care of this in recursion:
if the "next" player is the same as the current player,
and the ply count isn't incremented.
This does lead to unexpected increases in move calculation time
during mid-game, when a lot of bonus moves occur.
