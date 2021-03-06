# Kalah, an ancient game

Another implementation of [the game of Kalah](https://en.wikipedia.org/wiki/Kalah).
I implemented Wikipedia's rules exactly, as they were also what I was
accustomed to playing.

## Compiling

    $ git clone https://github.com/bediger4000/kalah.git $GOPATH/src/kalah
    $ go build kalah
    $ ./kalah
    OR
    $ ./kalah -M

You don't have to install it anywhere - it runs in place.
It has no configuration file(s).

The "-M" for Monte Carlo Tree Search is probably a
more exciting opponent.
The Alpha/Beta version just seems cold-blooded and relentless.

## Use

Players declare their moves with a number.
The usual 6 pits per player board is represented like this:

           computer

       5  4  3  2  1  0
    X                    Y
       0  1  2  3  4  5

            human

Computer's pot or store is X, human's is Y.
Players move by indicating which of their pits (0 through 5)
they want to move.
The program empties the chosen pit,
then distribute its contents (stones or seeds),
one per pit, traveling counterclockwise.
The program drops a stone in a player's own pot,
but not their opponents.
Players get a bonus move if they drop the final stone
in their hand into their own pot (X for computer or Y for human, above).
A bonus move can result in a bonus move.

This game does do captures.
If a player drops their final stone in an empty pit (0 through 5)
on their own side of the board,
that last stone,
and any stones in the pit opposite that empty pit get moved to the appropriate store.

`kalah` the program displays the current game board,
then asks the human to input a move, which is a single-digit
number, 0 through 5.

Command line flags:

    -C    Computer takes first move
    -M    Use MCTS instead of alpha/beta minimax
    -P    Do CPU profiling
    -R    Reverse printed board, top-to-bottom
    -U float
          UCTK factor, MCTS only (default 1)
    -d int
          lookahead depth for Alpha/Beta, moves for each side (default 6)
    -i int
          Number of iterations for MCTS (default 500000)
    -n int
          number of stones per pit (default 4)


"MCTS" means [Monte Carlo Tree Search](http://mcts.ai/).
It defaults to deciding what move to make by using Alpha/Beta minimaxing.

Reverse printed board makes it easier to open two terminals side-by-side
and play instances of the game against each other. Use "-R" on one of the
two instances so the programs print boards that look the same.

There's nothing magic about minimax using 6 move lookahead,
or Monte Carlo Tree Search using 500,000 iterations.
I found them empirically.
Both move choice algorithms can usually beat me if they move first,
or if I make a mistake,
and neither algorithm takes too long to decide with those values.

I don't see that varying the UCTK parameter makes any difference.
You can't use 0.0 as a value, however.

## Design

I used the Wikipedia article on [Alpha/Beta minimaxing](https://en.wikipedia.org/wiki/Alpha%E2%80%93beta_pruning).
I should have implemented one level of threading.
Static value calculated as difference of player's pots or stores.

The MCTS code follows the [MCTS example code](http://mcts.ai/code/python.html),
transliterated into Go for [squava](https://github.com/bediger4000/squava),
then mutated significantly to allow for Kalah's bonus moves.
It was suprisingly difficult to get bonus moves correct for MCTS.

### Bonus move

This variant has a bonus move.
Players get an extra move if they drop the last stone in their
hand into their own pot/store.
The Alpha/Beta minimaxing code takes care of this in recursion
by keeping the "next" player is the same as the current player,
and not incremented ply count.
It doesn't reach its move horizon while a player is in the middle
of making a multi-move sweep.
This does lead to unexpected increases in move calculation time
during mid-game, when a lot of bonus moves occur.
