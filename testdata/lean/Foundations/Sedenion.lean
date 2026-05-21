-- Sedenion associativity lemma.
-- Contains one sorry (work in progress).

namespace QBP

/-- Sedenion multiplication is not associative in general. -/
theorem sedenion_assoc (a b c : Sedenion) :
    a * b * c = a * (b * c) := by
  -- TODO: complete this proof
  sorry

end QBP
