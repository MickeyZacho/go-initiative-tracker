import { useState, useCallback } from "react";
import { apiGetArray } from "../lib/http";

export interface Encounter {
	ID: number;
	Name: string;
}

export function useEncounters(initialEncounterId?: number | null) {
	const [encounters, setEncounters] = useState<Encounter[]>([]);
	const [encounterId, setEncounterId] = useState<number>(0);
	const [error, setError] = useState<string>("");
	const [isLoading, setIsLoading] = useState<boolean>(false);

	const fetchEncounters = useCallback(
		async (
			fetchCharacters: (id: number) => Promise<void>,
			fetchLedger: (id: number) => Promise<void>,
		) => {
			setIsLoading(true);
			setError("");
			try {
				const data = await apiGetArray<Encounter>("/encounters");
				setEncounters(data);
				if (data.length > 0) {
					const preferredEncounterId =
						initialEncounterId &&
						data.some((enc) => enc.ID === initialEncounterId)
							? initialEncounterId
							: data[0].ID;
					setEncounterId(preferredEncounterId);
					await fetchCharacters(preferredEncounterId);
					await fetchLedger(preferredEncounterId);
				} else {
					setEncounterId(0);
				}
			} catch (err) {
				setError(
					err instanceof Error
						? err.message
						: "Failed to fetch encounters",
				);
				setEncounters([]);
			} finally {
				setIsLoading(false);
			}
		},
		[initialEncounterId],
	);

	return {
		encounters,
		encounterId,
		setEncounterId,
		fetchEncounters,
		error,
		isLoading,
	};
}
