// Package checkers is a pure-Go American checkers (English draughts) rules
// engine — no protocol, no I/O, standard library only.
//
// Extracted from https://github.com/richardwooding/kibitz
//
// Squares are the 32 dark squares, 0-indexed in PDN order: square 0 is the
// top-left dark square (Black's back row), row-major. Black sits on rows
// 0–2 (squares 0–11) and moves DOWN (increasing rows); White sits on rows
// 5–7 (squares 20–31) and moves up. Black moves first.
//
// Captures are forced: if any jump exists, only jump paths are legal, and a
// jump continues while the same piece can keep jumping. Promotion mid-path
// ends the move (a new king may not continue the jump sequence).
package checkers

import "errors"

// Cell values: 0 empty; +1 black man, +2 black king; -1 white man, -2 white king.
// (Positive = Black = P1, mirroring "P1 moves first".)
type Board [32]int8

// Side: 0 = Black (P1), 1 = White (P2).
type Side uint8

const (
	Black Side = 0
	White Side = 1
)

// Move is a path of square indices: [from, to] for a simple move, or
// [from, over₁-landing, over₂-landing, …] for a jump sequence.
type Move []int8

var ErrIllegalMove = errors.New("checkers: illegal move")

// Start is the standard opening position.
func Start() Board {
	var b Board
	for s := 0; s < 12; s++ {
		b[s] = 1 // black men
	}
	for s := 20; s < 32; s++ {
		b[s] = -1 // white men
	}
	return b
}

// row/col mapping for the dark-square indexing.
func rowOf(s int8) int8 { return s / 4 }
func colOf(s int8) int8 {
	r := s / 4
	if r%2 == 0 {
		return 2*(s%4) + 1
	}
	return 2 * (s % 4)
}

// sqAt maps (row, col) back to a square index, or -1 when off-board or a
// light square.
func sqAt(row, col int8) int8 {
	if row < 0 || row > 7 || col < 0 || col > 7 {
		return -1
	}
	if row%2 == 0 {
		if col%2 == 0 {
			return -1
		}
		return row*4 + (col-1)/2
	}
	if col%2 == 1 {
		return -1
	}
	return row*4 + col/2
}

func sideOf(v int8) (Side, bool) {
	switch {
	case v > 0:
		return Black, true
	case v < 0:
		return White, true
	default:
		return 0, false
	}
}

func isKing(v int8) bool { return v == 2 || v == -2 }

// rowDirs returns the movement row-directions for a piece.
func rowDirs(v int8) []int8 {
	if isKing(v) {
		return []int8{1, -1}
	}
	if v > 0 {
		return []int8{1} // black men move down
	}
	return []int8{-1} // white men move up
}

// promotionRow for the side moving.
func promotionRow(s Side) int8 {
	if s == Black {
		return 7
	}
	return 0
}

// LegalMoves generates all legal moves for side. If any capture exists,
// only (complete) capture paths are returned.
func LegalMoves(b Board, side Side) []Move {
	var jumps, simples []Move
	for s := int8(0); s < 32; s++ {
		pieceSide, occupied := sideOf(b[s])
		if !occupied || pieceSide != side {
			continue
		}
		jumpPaths(b, side, s, Move{s}, &jumps)
		if len(jumps) > 0 {
			continue // captures are forced; skip generating simples
		}
		for _, dr := range rowDirs(b[s]) {
			for _, dc := range []int8{-1, 1} {
				to := sqAt(rowOf(s)+dr, colOf(s)+dc)
				if to >= 0 && b[to] == 0 {
					simples = append(simples, Move{s, to})
				}
			}
		}
	}
	if len(jumps) > 0 {
		return jumps
	}
	return simples
}

// jumpPaths extends a capture sequence from the piece's current square,
// recording each maximal path.
func jumpPaths(b Board, side Side, at int8, path Move, out *[]Move) {
	extended := false
	for _, dr := range rowDirs(b[at]) {
		for _, dc := range []int8{-1, 1} {
			over := sqAt(rowOf(at)+dr, colOf(at)+dc)
			land := sqAt(rowOf(at)+2*dr, colOf(at)+2*dc)
			if over < 0 || land < 0 || b[land] != 0 {
				continue
			}
			overSide, occupied := sideOf(b[over])
			if !occupied || overSide == side {
				continue
			}
			// Take the jump on a copy.
			nb := b
			nb[land] = nb[at]
			nb[at] = 0
			nb[over] = 0
			newPath := append(append(Move{}, path...), land)
			// Promotion ends the move (American rule).
			if !isKing(nb[land]) && rowOf(land) == promotionRow(side) {
				*out = append(*out, newPath)
				extended = true
				continue
			}
			extended = true
			jumpPaths(nb, side, land, newPath, out)
		}
	}
	if !extended && len(path) > 1 {
		*out = append(*out, path)
	}
}

// Validate checks a move by membership in the legal set.
func Validate(b Board, side Side, m Move) error {
	for _, legal := range LegalMoves(b, side) {
		if movesEqual(legal, m) {
			return nil
		}
	}
	return ErrIllegalMove
}

func movesEqual(a, b Move) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Apply replays a (validated) move and returns the new board.
func Apply(b Board, side Side, m Move) Board {
	piece := b[m[0]]
	b[m[0]] = 0
	for i := 1; i < len(m); i++ {
		from, to := m[i-1], m[i]
		// A row distance of 2 is a jump: remove the captured piece.
		if abs8(rowOf(to)-rowOf(from)) == 2 {
			over := sqAt((rowOf(from)+rowOf(to))/2, (colOf(from)+colOf(to))/2)
			b[over] = 0
		}
	}
	last := m[len(m)-1]
	// Promote on reaching the far row.
	if !isKing(piece) && rowOf(last) == promotionRow(side) {
		piece *= 2
	}
	b[last] = piece
	return b
}

// Winner: the side to move loses when it has no legal moves (no pieces or
// fully blocked). Returns (winner, true) in that case.
func Winner(b Board, toMove Side) (Side, bool) {
	if len(LegalMoves(b, toMove)) == 0 {
		return 1 - toMove, true
	}
	return 0, false
}

func abs8(v int8) int8 {
	if v < 0 {
		return -v
	}
	return v
}
