# Kalah, an ancient game

Another implementation of [the game of Kalah](https://en.wikipedia.org/wiki/Kalah).
I implemented Wikipedia's rules exactly, as they were also what I was
accustomed to playing.

[Alpha/Beta minimaxing](https://en.wikipedia.org/wiki/Alpha%E2%80%93beta_pruning).
I should implemented one level of threading.

Static value calculated as difference of player's pots or stores.

### Bonus move

This variant has a bonus move.
Players get an extra move if they drop the last stone in their
hand into their own pot/store.
The code takes care of this by recursion:
the "next" player is the same as the current player,
and the ply count isn't incremented.
This does lead to unexpected increases in move calculation time
during mid-game, when a lot of bonus moves occur.
