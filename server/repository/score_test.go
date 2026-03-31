package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// scoreFor reproduces the scoring formula from InsertSubmissionAndUpdateScore
// without touching the database so we can unit-test it in isolation.
func scoreFor(isCorrect bool, timeTakenMs, timeLimit int) int {
	if !isCorrect {
		return 0
	}
	var score int
	if timeTakenMs < bonusThreshold {
		score = maxScore
	} else {
		score = maxScore * (timeLimit - timeTakenMs) / (timeLimit - bonusThreshold)
	}
	if score < 0 {
		score = 0
	}
	return score
}

// ---------------------------------------------------------------------------
// Table-driven scoring tests
// ---------------------------------------------------------------------------

func TestScoringFormula(t *testing.T) {
	const limit = 15_000 // 15 s in ms

	tests := []struct {
		name        string
		isCorrect   bool
		timeTakenMs int
		timeLimit   int
		wantScore   int
	}{
		{
			name:        "wrong answer always scores 0",
			isCorrect:   false,
			timeTakenMs: 0,
			timeLimit:   limit,
			wantScore:   0,
		},
		{
			name:        "correct answer under bonus threshold gives full score",
			isCorrect:   true,
			timeTakenMs: bonusThreshold - 1, // 2999 ms
			timeLimit:   limit,
			wantScore:   maxScore, // 1000
		},
		{
			name:        "correct answer exactly at bonus threshold gives full score",
			isCorrect:   true,
			timeTakenMs: bonusThreshold, // 3000 ms exactly → NOT < threshold
			timeLimit:   limit,
			// score = 1000 * (15000 - 3000) / (15000 - 3000) = 1000
			wantScore: maxScore,
		},
		{
			name:        "correct answer at mid-point gives half score",
			isCorrect:   true,
			timeTakenMs: 9_000, // halfway between 3000 and 15000
			timeLimit:   limit,
			// score = 1000 * (15000 - 9000) / (15000 - 3000) = 1000 * 6000 / 12000 = 500
			wantScore: 500,
		},
		{
			name:        "correct answer just before time limit scores near zero",
			isCorrect:   true,
			timeTakenMs: limit - 1, // 14999 ms
			timeLimit:   limit,
			// score = 1000 * 1 / 12000 = 0 (integer division)
			wantScore: 0,
		},
		{
			name:        "correct answer at time limit gives 0 (formula = 0)",
			isCorrect:   true,
			timeTakenMs: limit,
			timeLimit:   limit,
			// score = 1000 * 0 / 12000 = 0
			wantScore: 0,
		},
		{
			name:        "correct answer beyond time limit clamps to 0",
			isCorrect:   true,
			timeTakenMs: limit + 1000,
			timeLimit:   limit,
			// score = 1000 * (-1000) / 12000 = negative → clamped to 0
			wantScore: 0,
		},
		{
			name:        "wrong answer with very fast time still scores 0",
			isCorrect:   false,
			timeTakenMs: 500,
			timeLimit:   limit,
			wantScore:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scoreFor(tc.isCorrect, tc.timeTakenMs, tc.timeLimit)
			assert.Equal(t, tc.wantScore, got,
				"scoreFor(isCorrect=%v, taken=%d, limit=%d)", tc.isCorrect, tc.timeTakenMs, tc.timeLimit)
		})
	}
}

// ---------------------------------------------------------------------------
// Boundary: maxScore / bonusThreshold package-level vars
// ---------------------------------------------------------------------------

func TestScoringConstants(t *testing.T) {
	assert.Equal(t, 1000, maxScore, "maxScore must be 1000")
	assert.Equal(t, 3000, bonusThreshold, "bonusThreshold must be 3000 ms")
}

func TestScoringFormula_NeverExceedsMaxScore(t *testing.T) {
	// For any timeTaken >= 0 the score must never exceed maxScore
	for _, taken := range []int{0, 1, 100, 1000, 2999, 3000, 5000, 10000, 15000, 20000} {
		got := scoreFor(true, taken, 15_000)
		assert.LessOrEqual(t, got, maxScore,
			"score should never exceed maxScore; taken=%d", taken)
	}
}

func TestScoringFormula_NeverNegative(t *testing.T) {
	for _, taken := range []int{0, 15001, 99999} {
		got := scoreFor(true, taken, 15_000)
		assert.GreaterOrEqual(t, got, 0, "score must not be negative; taken=%d", taken)
	}
}
