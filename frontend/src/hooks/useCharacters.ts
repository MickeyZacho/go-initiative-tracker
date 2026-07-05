import { useState, useCallback } from "react";
import { apiGetArray } from "../lib/http";
import type { Character } from "../components/CharacterList";

export function useCharacters() {
	const [characters, setCharacters] = useState<Character[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [error, setError] = useState<string>("");

	const fetchCharacters = useCallback(async (encId: number) => {
		setIsLoading(true);
		setError("");
		try {
			// Tell the backend which encounter is selected (fire-and-forget),
			// then load that encounter's characters.
			await fetch("/api/select-encounter", {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ id: encId }),
			});
			const data = await apiGetArray<Character>(
				`/characters?encounter_id=${encId}`,
			);
			setCharacters(data);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to fetch characters",
			);
			setCharacters([]);
		} finally {
			setIsLoading(false);
		}
	}, []);

	return {
		characters,
		setCharacters,
		fetchCharacters,
		isLoading,
		error,
	};
}
