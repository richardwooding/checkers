package checkers

import (
	"errors"
	"testing"
)

// place builds a board from explicit piece placements.
func place(pieces map[int8]int8) Board {
	var b Board
	for sq, v := range pieces {
		b[sq] = v
	}
	return b
}

func TestStartPosition(t *testing.T) {
	b := Start()
	moves := LegalMoves(b, Black)
	// Black's opening: each of the 4 front-row men (squares 8..11) has up to
	// 2 diagonal moves; edges have 1. Standard count is 7.
	if len(moves) != 7 {
		t.Fatalf("black opening moves = %d, want 7 (%v)", len(moves), moves)
	}
	for _, m := range moves {
		if len(m) != 2 {
			t.Fatalf("opening move with jump: %v", m)
		}
	}
	// White mirrors.
	if got := len(LegalMoves(b, White)); got != 7 {
		t.Fatalf("white opening moves = %d", got)
	}
}

func TestForcedCapture(t *testing.T) {
	// Black man on 13 (row 3); white man on 17 (row 4) diagonally adjacent;
	// landing square beyond is empty → black MUST jump; the quiet moves of
	// another black man must not appear.
	b := place(map[int8]int8{13: 1, 17: -1, 0: 1})
	moves := LegalMoves(b, Black)
	if len(moves) == 0 {
		t.Fatal("no moves generated")
	}
	for _, m := range moves {
		if len(m) < 2 || abs8(rowOf(m[1])-rowOf(m[0])) != 2 {
			t.Fatalf("non-capture offered while capture exists: %v", m)
		}
	}
	// The quiet move 13→17-ish must be rejected outright.
	if err := Validate(b, Black, Move{0, 5}); !errors.Is(err, ErrIllegalMove) {
		t.Fatalf("quiet move accepted during forced capture: %v", err)
	}
}

func TestMultiJumpForced(t *testing.T) {
	// Geometry (verified against the sq/row/col mapping): black at 9=(r2,c3)
	// jumps white at 13=(r3,c2) landing 16=(r4,c1); from there white at
	// 21=(r5,c2) is jumpable landing 25=(r6,c3) → double jump 9→16→25.
	b := place(map[int8]int8{9: 1, 13: -1, 21: -1})
	moves := LegalMoves(b, Black)
	if len(moves) == 0 {
		t.Fatal("no moves")
	}
	for _, m := range moves {
		t.Logf("move: %v", m)
	}
	// Find the double jump; single jump alone must NOT be legal.
	foundDouble := false
	for _, m := range moves {
		if len(m) == 3 {
			foundDouble = true
			// Stopping after the first hop is illegal.
			if err := Validate(b, Black, m[:2]); !errors.Is(err, ErrIllegalMove) {
				t.Fatalf("truncated jump accepted: %v", err)
			}
			// Applying removes both captured men.
			nb := Apply(b, Black, m)
			whites := 0
			for _, v := range nb {
				if v < 0 {
					whites++
				}
			}
			if whites != 0 {
				t.Fatalf("captures not applied: %v", nb)
			}
		}
	}
	if !foundDouble {
		t.Fatalf("double jump not generated: %v", moves)
	}
}

func TestPromotionEndsJump(t *testing.T) {
	// Black at 21=(r5,c2) jumps white at 25=(r6,c3) landing on 30=(r7,c4),
	// the crown row. White at 26=(r6,c5) would be jumpable by a KING from 30
	// — but promotion ends the move, so no path may continue past row 7.
	b := place(map[int8]int8{21: 1, 25: -1, 26: -1})
	moves := LegalMoves(b, Black)
	for _, m := range moves {
		last := m[len(m)-1]
		if rowOf(last) == 7 && len(m) > 2 {
			t.Fatalf("jump continued past promotion: %v", m)
		}
	}
	// Apply the promoting jump; the piece must be a king.
	var promo Move
	for _, m := range moves {
		if rowOf(m[len(m)-1]) == 7 {
			promo = m
			break
		}
	}
	if promo == nil {
		t.Fatalf("no promoting jump found in %v", moves)
	}
	nb := Apply(b, Black, promo)
	if nb[promo[len(promo)-1]] != 2 {
		t.Fatalf("piece not crowned: %v", nb[promo[len(promo)-1]])
	}
}

func TestKingMovesBothWays(t *testing.T) {
	b := place(map[int8]int8{13: 2}) // lone black king mid-board
	moves := LegalMoves(b, Black)
	if len(moves) != 4 {
		t.Fatalf("king moves = %d, want 4 (%v)", len(moves), moves)
	}
	up, down := false, false
	for _, m := range moves {
		if rowOf(m[1]) > rowOf(m[0]) {
			down = true
		} else {
			up = true
		}
	}
	if !up || !down {
		t.Fatal("king restricted to one direction")
	}
}

func TestManCannotMoveBackward(t *testing.T) {
	b := place(map[int8]int8{13: 1})
	for _, m := range LegalMoves(b, Black) {
		if rowOf(m[1]) <= rowOf(m[0]) {
			t.Fatalf("black man moved backward: %v", m)
		}
	}
}

func TestBlockedSideLoses(t *testing.T) {
	// White man on 28 (row 7 corner region) hemmed in by black pieces so it
	// cannot move or jump; white to move → black wins.
	// Square 28: row 7, col? row7 odd → col = 2*(28%4)=0. Diagonals up:
	// (6, 1) = sq 24? row6 even: col1 → 24. Jump landing (5,2): row5 odd
	// col2 → 21. Block 24 with black and 21 with black (occupied landing).
	b := place(map[int8]int8{28: -1, 24: 1, 21: 1})
	winner, over := Winner(b, White)
	if !over || winner != Black {
		t.Fatalf("winner=%v over=%v", winner, over)
	}
	// Black to move has moves — not over.
	if _, over := Winner(b, Black); over {
		t.Fatal("black wrongly out of moves")
	}
}

func TestNoPiecesLoses(t *testing.T) {
	b := place(map[int8]int8{5: 1})
	if winner, over := Winner(b, White); !over || winner != Black {
		t.Fatalf("white with no pieces should lose (winner=%v over=%v)", winner, over)
	}
}

func TestJumpsCannotRevisitCapturedPiece(t *testing.T) {
	// A circular-ish setup: ensure a captured piece is gone during path
	// extension (the DFS mutates a board copy). Black king on 13 with white
	// men placed so a cycle would revisit square 17's piece.
	b := place(map[int8]int8{13: 2, 17: -1, 25: -1, 26: -1, 18: -1})
	for _, m := range LegalMoves(b, Black) {
		seenOver := map[int8]bool{}
		for i := 1; i < len(m); i++ {
			if abs8(rowOf(m[i])-rowOf(m[i-1])) == 2 {
				over := sqAt((rowOf(m[i-1])+rowOf(m[i]))/2, (colOf(m[i-1])+colOf(m[i]))/2)
				if seenOver[over] {
					t.Fatalf("move %v jumps square %d twice", m, over)
				}
				seenOver[over] = true
			}
		}
	}
}
