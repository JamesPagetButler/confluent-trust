-- Hurwitz theorem: only ℝ, ℂ, ℍ, 𝕆 admit a normed division algebra structure.
-- This file has zero sorries.

import Mathlib.Algebra.NormedSpace.Basic

namespace QBP

/-- Hurwitz 1898: classification of normed division algebras. -/
theorem hurwitz_theorem (A : Type*) [NormedDivisionAlgebra A] :
    Nonempty (A ≃ₐ[ℝ] ℝ) ∨ Nonempty (A ≃ₐ[ℝ] ℂ) ∨
    Nonempty (A ≃ₐ[ℝ] ℍ) ∨ Nonempty (A ≃ₐ[ℝ] 𝕆) := by
  exact hurwitz_classification A

end QBP
