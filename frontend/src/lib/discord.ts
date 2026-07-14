// Build a Discord CDN avatar URL from a user's id and avatar hash. Returns null
// when either is missing so callers can fall back to a placeholder.
export function discordAvatarUrl(
	discordID: string,
	avatar: string,
): string | null {
	if (!discordID || !avatar) {
		return null;
	}
	return `https://cdn.discordapp.com/avatars/${discordID}/${avatar}.png`;
}
