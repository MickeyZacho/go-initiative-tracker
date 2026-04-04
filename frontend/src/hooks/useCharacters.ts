import { useState, useCallback } from "react";
import { apiUrl } from "../lib/api";
import { parseJsonResponse } from "../lib/http";
import type { Character } from "../components/CharacterList";

export function useCharacters() {
	const [characters, setCharacters] = useState<Character[]>([]);
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [error, setError] = useState<string>("");

	const fetchCharacters = useCallback(async (encId: number) => {
		setIsLoading(true);
		setError("");
		try {
			await fetch(apiUrl("/api/select-encounter"), {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ id: encId }),
			});
			const response = await fetch(
				apiUrl(`/api/characters?encounter_id=${encId}`),
				{ credentials: "include" },
			);
			if (!response.ok) {
				throw new Error("Failed to fetch characters");
			}
			const payload = await parseJsonResponse<unknown>(response);
			const data: Character[] = Array.isArray(payload) ? payload : [];
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
