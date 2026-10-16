package shaman

import (
	"math"
	"testing"

	"github.com/wowsims/forever/sim/core"
)

// Rank 5 hits through 408428: build 70425's 239 + 1.3 a level to 57 at 0.143 (was 403 + 3.4 at 0.214).
func TestFireNovaDamage(t *testing.T) {
	e := fireNovaDamage.DamageEffect()
	if avg := e.Average(core.CharacterLevel); math.Abs(avg-245) > 0.01 || math.Abs(e.Coeff()-0.143) > 1e-6 {
		t.Fatalf("Fire Nova rank 5 averages %v at %v, want 245 at 0.143", avg, e.Coeff())
	}
}
