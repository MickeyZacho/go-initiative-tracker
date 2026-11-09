import type { Character } from "../components/CharacterList";

// Mirrors the backend's player view (backend/player_view.go) so the DM can
// preview exactly what players see. Players never rely on this: their data
// arrives already redacted from the server.

export type HealthStatus = "Healthy" | "Wounded" | "Bloodied" | "Down";

export function healthStatus(current: number, max: number): HealthStatus {
	if (current <= 0) return "Down";
	if (max > 0 && current * 2 <= max) return "Bloodied";
	if (current < max) return "Wounded";
	return "Healthy";
}

export function redactForPlayers(characters: Character[]): Character[] {
	return characters.map((c) =>
		c.Type !== "npc" || c.Redacted
			? c
			: {
					...c,
					Redacted: true,
					HealthStatus: healthStatus(c.CurrentHP, c.MaxHP),
					ArmorClass: 0,
					ToHitModifier: 0,
					CurrentHP: 0,
					MaxHP: 0,
				},
	);
}
