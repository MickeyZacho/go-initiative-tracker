import { useState, useEffect, useCallback } from "react";
import { CharacterRow } from "./CharacterRow";
import {
	Card,
	CardContent,
	Typography,
	Box,
	Button,
	Select,
	MenuItem,
	FormControl,
	InputLabel,
	Stack,
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material/Select";

export interface Character {
	ID: number;
	Name: string;
	ArmorClass: number;
	ToHitModifier?: number;
	MaxHP: number;
	CurrentHP: number;
	Initiative: number;
	IsActive: boolean;
	OwnerID: string;
}

interface Encounter {
	ID: number;
	Name: string;
}

export const CharacterList: React.FC = () => {
	const [encounters, setEncounters] = useState<Encounter[]>([]);
	const [encounterId, setEncounterId] = useState<number>(0);
	const [characters, setCharacters] = useState<Character[]>([]);
	const [, setSelected] = useState<number | null>(null);
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [error, setError] = useState<string>("");

	const fetchCharacters = useCallback(async (encId: number) => {
		setIsLoading(true);
		setError("");
		try {
			await fetch("/api/select-encounter", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ id: encId }),
			});
			const response = await fetch(
				`/api/characters?encounter_id=${encId}`,
			);
			if (!response.ok) {
				throw new Error("Failed to fetch characters");
			}
			const payload = await response.json();
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

	const fetchEncounters = useCallback(async () => {
		setIsLoading(true);
		setError("");
		try {
			const response = await fetch("/api/encounters");
			if (!response.ok) {
				throw new Error("Failed to fetch encounters");
			}
			const payload = await response.json();
			const data: Encounter[] = Array.isArray(payload) ? payload : [];
			setEncounters(data);
			if (data.length > 0) {
				setEncounterId(data[0].ID);
				await fetchCharacters(data[0].ID);
			} else {
				setEncounterId(0);
				setCharacters([]);
			}
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to fetch encounters",
			);
			setEncounters([]);
			setCharacters([]);
		} finally {
			setIsLoading(false);
		}
	}, [fetchCharacters]);

	useEffect(() => {
		fetchEncounters();
	}, [fetchEncounters]);

	const handleEncounterChange = async (event: SelectChangeEvent) => {
		const newId = Number(event.target.value);
		setEncounterId(newId);
		await fetchCharacters(newId);
	};

	const addCharacter = () => {
		const newId = Date.now();
		setCharacters((prev) => [
			...prev,
			{
				ID: newId,
				Name: "",
				ArmorClass: 10,
				ToHitModifier: 0,
				MaxHP: 10,
				CurrentHP: 10,
				Initiative: 0,
				IsActive: false,
				OwnerID: "",
			},
		]);
	};

	const saveCharacter = async (character: Character) => {
		const idToSend = character.ID > 2147483647 ? 0 : character.ID;
		const response = await fetch("/save-character", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				id: idToSend,
				name: character.Name,
				armorClass: Number(character.ArmorClass),
				maxHP: Number(character.MaxHP),
				currentHP: Number(character.CurrentHP),
				initiative: Number(character.Initiative),
				toHitModifier: Number(character.ToHitModifier ?? 0),
				isActive: Boolean(character.IsActive),
			}),
		});
		if (!response.ok) {
			throw new Error("Failed to save character");
		}
		await fetchCharacters(encounterId);
	};

	const removeCharacter = async (characterID: number) => {
		const response = await fetch("/remove-character-from-encounter", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ character_id: characterID }),
		});
		const data = await response.json();
		if (!response.ok || data.status !== "success") {
			throw new Error(data.message || "Failed to remove character");
		}
		await fetchCharacters(encounterId);
	};

	const nextCharacter = () => {
		if (characters.length === 0) return;
		// Always use sorted list for cycling
		const sorted = [...characters].sort(
			(a, b) => b.Initiative - a.Initiative,
		);
		const currentIdx = sorted.findIndex((c) => c.IsActive);
		let nextIdx = currentIdx + 1;
		if (currentIdx === -1 || nextIdx >= sorted.length) nextIdx = 0;
		const nextId = sorted[nextIdx].ID;
		// Update IsActive based on sorted order, but preserve original array order
		const updated = characters.map((c) => ({
			...c,
			IsActive: c.ID === nextId,
		}));
		setCharacters(updated);
		setSelected(nextId);
	};

	// Spacebar triggers nextCharacter
	useEffect(() => {
		const handleKeyDown = (e: KeyboardEvent) => {
			if (
				e.code === "Space" &&
				!(
					e.target instanceof HTMLInputElement ||
					e.target instanceof HTMLTextAreaElement
				)
			) {
				e.preventDefault();
				nextCharacter();
			}
		};
		window.addEventListener("keydown", handleKeyDown);
		return () => window.removeEventListener("keydown", handleKeyDown);
	}, [nextCharacter, characters, encounterId]);

	return (
		<Box display="flex" justifyContent="center" alignItems="center">
			<Card
				sx={{
					width: 700,
					borderRadius: 3,
					boxShadow: 3,
					p: 3,
				}}
			>
				<CardContent>
					<Stack spacing={3}>
						{/* Encounter Switcher */}
						<FormControl fullWidth variant="outlined">
							<InputLabel id="encounter-label">
								Encounter
							</InputLabel>
							<Select
								labelId="encounter-label"
								value={String(encounterId)}
								label="Encounter"
								onChange={handleEncounterChange}
								sx={{ fontWeight: 500, color: "#1976d2" }}
							>
								{encounters.map((enc) => (
									<MenuItem
										key={enc.ID}
										value={String(enc.ID)}
									>
										{enc.Name}
									</MenuItem>
								))}
							</Select>
						</FormControl>
						{error && (
							<Typography color="error">{error}</Typography>
						)}
						<Typography
							variant="h4"
							color="primary"
							fontWeight={700}
							letterSpacing={1}
						>
							Characters
						</Typography>
						<Stack spacing={2}>
							{[...characters]
								.sort((a, b) => b.Initiative - a.Initiative)
								.map((character) => (
									<div
										style={{ width: "100%" }}
										key={character.ID}
									>
										<CharacterRow
											character={character}
											setCharacters={setCharacters}
											setSelected={setSelected}
										/>
										<Stack
											direction="row"
											spacing={1}
											mt={1}
										>
											<Button
												size="small"
												variant="contained"
												onClick={() =>
													saveCharacter(character)
												}
											>
												Save
											</Button>
											<Button
												size="small"
												color="error"
												variant="outlined"
												onClick={() =>
													removeCharacter(
														character.ID,
													)
												}
											>
												Remove
											</Button>
										</Stack>
									</div>
								))}
						</Stack>
						<Stack
							direction="row"
							spacing={2}
							justifyContent="center"
							alignItems="center"
							mt={2}
						>
							<Button
								variant="contained"
								color="success"
								onClick={addCharacter}
								sx={{
									fontWeight: 600,
									fontSize: "1rem",
									px: 3,
									py: 1,
								}}
							>
								Add Character
							</Button>
							<Button
								variant="contained"
								color="primary"
								onClick={nextCharacter}
								disabled={isLoading}
								sx={{
									fontWeight: 600,
									fontSize: "1rem",
									px: 3,
									py: 1,
								}}
							>
								Next Character
							</Button>
						</Stack>
						{isLoading && (
							<Typography color="text.secondary">
								Loading...
							</Typography>
						)}
					</Stack>
				</CardContent>
			</Card>
		</Box>
	);
};
