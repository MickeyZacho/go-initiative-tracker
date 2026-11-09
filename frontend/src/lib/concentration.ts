// Constitution save DC to keep concentrating after taking damage (5e):
// 10 or half the damage taken, whichever is higher (capped at 30 in the 2024
// rules). One save per source of damage.
export function concentrationDC(damage: number): number {
	return Math.min(30, Math.max(10, Math.floor(damage / 2)));
}
