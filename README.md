# checkers

A pure-Go American checkers (English draughts) rules engine — no protocol,
no I/O, standard library only.

`LegalMoves` generates every legal move as a *path* of squares; captures are
forced (if any jump exists, only jump paths are returned), multi-jumps
extend until no further jump is possible, and promotion ends a jump
sequence. `Validate` checks a submitted path by membership, and `Winner`
reports a loss for the side with no legal move — ideal for
both-sides-validate multiplayer where neither client is trusted.

```go
b := checkers.Start()
moves := checkers.LegalMoves(b, checkers.Black) // Black moves first
if err := checkers.Validate(b, checkers.Black, moves[0]); err == nil {
    b = checkers.Apply(b, checkers.Black, moves[0])
}
if winner, over := checkers.Winner(b, checkers.White); over {
    _ = winner
}
```

Board is the 32 dark squares in PDN order; a `Move` is a path
(`[from, to]` for a slide, `[from, land1, land2, …]` for a jump chain).

## Install

```sh
go get github.com/richardwooding/checkers
```

Extracted from [kibitz](https://github.com/richardwooding/kibitz).

## License

MIT
