-- Phantom proof file: theorem appears here but anchor says not_started.
-- This file triggers Invariant 5 (phantom-artifact rule).

namespace QBP

/-- This theorem exists in the file but the anchor marks it as not_started. -/
theorem phantom_proof (n : ℕ) : n + 0 = n := by
  ring

end QBP
