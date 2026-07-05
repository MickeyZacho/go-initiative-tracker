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
			// The encounter is passed explicitly per request; the backend holds
			// no selected-encounter state.
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
