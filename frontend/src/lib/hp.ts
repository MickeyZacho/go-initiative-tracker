// HP entry for the combat row. DMs think in deltas ("the goblin takes 7"), so
// the HP field accepts:
//   "-7"  → damage: temp HP absorbs it first, the rest comes off current HP
//   "+5"  → healing: add to current HP (never to temp HP)
//   "12"  → set current HP to 12 (temp HP untouched)
// Current HP is clamped to 0..maxHP. Returns null when the input isn't a
// number, so the caller can treat it as "no change".
export interface HpState {
	current: number;
	temp: number;
}

export function applyHpInput(
	input: string,
	state: HpState,
	maxHP: number,
): HpState | null {
	const text = input.trim().replace(/\s+/g, "");
	const match = /^([+-]?)(\d+)$/.exec(text);
	if (!match) return null;
	const [, sign, digits] = match;
	const amount = Number(digits);
	const clamp = (n: number) => Math.max(0, Math.min(maxHP, n));

	if (sign === "-") {
		const absorbed = Math.min(state.temp, amount);
		return {
			temp: state.temp - absorbed,
			current: clamp(state.current - (amount - absorbed)),
		};
	}
	if (sign === "+") {
		return { temp: state.temp, current: clamp(state.current + amount) };
	}
	return { temp: state.temp, current: clamp(amount) };
}
