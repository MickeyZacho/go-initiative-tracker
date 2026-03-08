import { useState, useEffect, useCallback } from "react";
import { CharacterRow } from "./CharacterRow";
import { parseJsonResponse } from "../lib/http";
import { apiUrl } from "../lib/api";
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

interface CharacterListProps {
	initialEncounterId?: number | null;
}

export const CharacterList: React.FC<CharacterListProps> = ({
	initialEncounterId,
}) => {
	const [encounters, setEncounters] = useState<Encounter[]>([]);
	const [encounterId, setEncounterId] = useState<number>(0);
	const [characters, setCharacters] = useState<Character[]>([]);
	const [combatStartedByEncounter, setCombatStartedByEncounter] = useState<
		Record<number, boolean>
	>({});
	const [libraryCharacters, setLibraryCharacters] = useState<Character[]>([]);
	const [selectedAddCharacterId, setSelectedAddCharacterId] =
		useState<number>(0);
	const [, setSelected] = useState<number | null>(null);
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [error, setError] = useState<string>("");
	const combatStarted =
		encounterId > 0
			? Boolean(combatStartedByEncounter[encounterId])
			: false;

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

	const fetchEncounters = useCallback(async () => {
		setIsLoading(true);
		setError("");
		try {
			const response = await fetch(apiUrl("/api/encounters"), {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to fetch encounters");
			}
			const payload = await parseJsonResponse<unknown>(response);
			const data: Encounter[] = Array.isArray(payload) ? payload : [];
			setEncounters(data);
			if (data.length > 0) {
				const preferredEncounterId =
					initialEncounterId &&
					data.some((enc) => enc.ID === initialEncounterId)
						? initialEncounterId
						: data[0].ID;
				setEncounterId(preferredEncounterId);
				await fetchCharacters(preferredEncounterId);
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
	}, [fetchCharacters, initialEncounterId]);

	const fetchLibraryCharacters = useCallback(async () => {
		try {
			const response = await fetch(apiUrl("/api/characters/library"), {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to fetch character library");
			}
			const payload = await parseJsonResponse<unknown>(response);
			const data: Character[] = Array.isArray(payload) ? payload : [];
			setLibraryCharacters(data);
			if (data.length > 0) {
				setSelectedAddCharacterId(data[0].ID);
			}
		} catch {
			setLibraryCharacters([]);
		}
	}, []);

	useEffect(() => {
		fetchEncounters();
		fetchLibraryCharacters();
	}, [fetchEncounters, fetchLibraryCharacters]);

	useEffect(() => {
		const available = libraryCharacters.filter(
			(libChar) =>
				!characters.some((encChar) => encChar.ID === libChar.ID),
		);
		if (available.length === 0) {
			setSelectedAddCharacterId(0);
			return;
		}
		if (!available.some((c) => c.ID === selectedAddCharacterId)) {
			setSelectedAddCharacterId(available[0].ID);
		}
	}, [libraryCharacters, characters, selectedAddCharacterId]);

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
		const response = await fetch(apiUrl("/save-character"), {
			method: "POST",
			credentials: "include",
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
		const response = await fetch(
			apiUrl("/remove-character-from-encounter"),
			{
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					encounter_id: encounterId,
					character_id: characterID,
				}),
			},
		);
		const data = await parseJsonResponse<{
			status?: string;
			message?: string;
		}>(response);
		if (!response.ok || data.status !== "success") {
			throw new Error(data.message || "Failed to remove character");
		}
		await fetchCharacters(encounterId);
	};

	const addExistingCharacterToEncounter = async () => {
		if (!encounterId || !selectedAddCharacterId) {
			return;
		}
		setError("");
		try {
			const response = await fetch(
				apiUrl("/add-character-to-encounter"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						encounter_id: encounterId,
						character_id: selectedAddCharacterId,
					}),
				},
			);
			const data = await parseJsonResponse<{
				status?: string;
				message?: string;
			}>(response);
			if (!response.ok || data.status !== "success") {
				throw new Error(
					data.message || "Failed to add character to encounter",
				);
			}
			await fetchCharacters(encounterId);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to add character to encounter",
			);
		}
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

	const handleStartCombat = () => {
		if (!encounterId || characters.length === 0) return;
		setCombatStartedByEncounter((prev) => ({
			...prev,
			[encounterId]: true,
		}));
	};

	const handleBackToSetup = () => {
		if (!encounterId) return;
		setCombatStartedByEncounter((prev) => ({
			...prev,
			[encounterId]: false,
		}));
	};

	const availableLibraryCharacters = libraryCharacters.filter(
		(libChar) => !characters.some((encChar) => encChar.ID === libChar.ID),
	);

	// Spacebar triggers nextCharacter
	useEffect(() => {
		const handleKeyDown = (e: KeyboardEvent) => {
			if (!combatStarted) return;
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
	}, [nextCharacter, characters, encounterId, combatStarted]);

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
						<Typography
							color={
								combatStarted
									? "success.main"
									: "text.secondary"
							}
						>
							Status: {combatStarted ? "In Combat" : "Setup"}
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
							<FormControl sx={{ minWidth: 240 }}>
								<InputLabel id="add-existing-character-label">
									Add Existing Character
								</InputLabel>
								<Select
									labelId="add-existing-character-label"
									value={
										selectedAddCharacterId
											? String(selectedAddCharacterId)
											: ""
									}
									label="Add Existing Character"
									onChange={(event: SelectChangeEvent) =>
										setSelectedAddCharacterId(
											Number(event.target.value),
										)
									}
								>
									{availableLibraryCharacters.map(
										(libChar) => (
											<MenuItem
												key={libChar.ID}
												value={String(libChar.ID)}
											>
												{libChar.Name}
											</MenuItem>
										),
									)}
								</Select>
							</FormControl>
							<Button
								variant="outlined"
								onClick={addExistingCharacterToEncounter}
								disabled={
									!encounterId ||
									!selectedAddCharacterId ||
									availableLibraryCharacters.length === 0
								}
							>
								Add to Encounter
							</Button>
							<Button
								variant={
									combatStarted ? "outlined" : "contained"
								}
								color={combatStarted ? "inherit" : "warning"}
								onClick={
									combatStarted
										? handleBackToSetup
										: handleStartCombat
								}
								disabled={
									!encounterId ||
									(!combatStarted && characters.length === 0)
								}
							>
								{combatStarted
									? "Back to Setup"
									: "Start Combat"}
							</Button>
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
								disabled={
									isLoading ||
									!combatStarted ||
									characters.length === 0
								}
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
