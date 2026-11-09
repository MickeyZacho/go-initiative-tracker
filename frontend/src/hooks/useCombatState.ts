import { useCallback, useState } from "react";
import { apiGet } from "../lib/http";

// useCombatState tracks encounter-level combat state that isn't on the
// character rows — the round (0 = combat not started).
export function useCombatState() {
	const [round, setRound] = useState(0);

	const fetchCombatState = useCallback(async (encId: number) => {
		if (!encId) {
			setRound(0);
			return;
		}
		try {
			const res = await apiGet<{ round?: number }>(
				`/encounters/combat/state?encounter_id=${encId}`,
			);
			setRound(typeof res.round === "number" ? res.round : 0);
		} catch {
			setRound(0);
		}
	}, []);

	return { round, fetchCombatState };
}
