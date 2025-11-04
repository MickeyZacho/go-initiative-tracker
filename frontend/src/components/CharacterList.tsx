import { useState, useEffect } from "react";
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
	MaxHP: number;
	CurrentHP: number;
	Initiative: number;
	IsActive: boolean;
	OwnerID: string;
}

interface CharacterListProps {
	initialCharacters: Character[];
	enemies: Character[];
}

export const CharacterList: React.FC<CharacterListProps> = ({
	initialCharacters,
	enemies,
}) => {
	// Enemy selector state
	const [selectedEnemyId, setSelectedEnemyId] = useState<string>(
		enemies[0]?.ID ? String(enemies[0].ID) : ""
	);
	// Encounter switcher state (demo encounters)
	const [encounterId, setEncounterId] = useState<number>(1);
	const encounters = [
		{ id: 1, name: "Goblin Ambush" },
		{ id: 2, name: "Dragon's Lair" },
		{ id: 3, name: "Bandit Camp" },
	];

	// Characters state is now encounter-specific
	const [charactersByEncounter, setCharactersByEncounter] = useState<
		Record<number, Character[]>
	>({
		1: initialCharacters,
		2: initialCharacters.slice(0, 2),
		3: initialCharacters.slice(2),
	});
	const characters = charactersByEncounter[encounterId] || [];
	const [, setSelected] = useState<number | null>(null);

	// Placeholder for fetching characters for an encounter
	const fetchCharacters = (encId: number) => {
		// TODO: Replace with real API call
		// Simulate fetch by keeping state for now
		// setCharactersByEncounter({ ...charactersByEncounter, [encId]: fetchedCharacters });
	};

	// When encounter changes, fetch characters (future-proof)
	const handleEncounterChange = (event: SelectChangeEvent) => {
		const newId = Number(event.target.value);
		setEncounterId(newId);
		fetchCharacters(newId);
	};

	const addCharacter = () => {
		const newId = Date.now();
		const updated = [
			...characters,
			{
				ID: newId,
				Name: "",
				ArmorClass: 0,
				MaxHP: 0,
				CurrentHP: 0,
				Initiative: 0,
				IsActive: false,
				OwnerID: "",
			},
		];
		setCharactersByEncounter((prev) => ({
			...prev,
			[encounterId]: updated,
		}));
	};

	const addEnemyToEncounter = () => {
		const enemy = enemies.find((e) => String(e.ID) === selectedEnemyId);
		if (!enemy) return;
		// Clone enemy with new ID to avoid duplicates
		const newEnemy = { ...enemy, ID: Date.now() };
		setCharactersByEncounter((prev) => ({
			...prev,
			[encounterId]: [...characters, newEnemy],
		}));
	};

	const nextCharacter = () => {
		if (characters.length === 0) return;
		// Always use sorted list for cycling
		const sorted = [...characters].sort(
			(a, b) => b.Initiative - a.Initiative
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
		setCharactersByEncounter((prev) => ({
			...prev,
			[encounterId]: updated,
		}));
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
										key={enc.id}
										value={String(enc.id)}
									>
										{enc.name}
									</MenuItem>
								))}
							</Select>
						</FormControl>
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
											setCharacters={(chars) =>
												setCharactersByEncounter(
													(prev) => ({
														...prev,
														[encounterId]:
															typeof chars ===
															"function"
																? chars(
																		prev[
																			encounterId
																		] || []
																  )
																: chars,
													})
												)
											}
											setSelected={setSelected}
										/>
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
						<Stack
							direction="row"
							spacing={2}
							justifyContent="center"
							alignItems="center"
							mt={2}
						>
							<FormControl
								variant="outlined"
								sx={{ minWidth: 120 }}
							>
								<InputLabel id="enemy-label">Enemy</InputLabel>
								<Select
									labelId="enemy-label"
									value={selectedEnemyId}
									label="Enemy"
									onChange={(event: SelectChangeEvent) =>
										setSelectedEnemyId(event.target.value)
									}
									sx={{ color: "#d32f2f" }}
								>
									{enemies.map((e) => (
										<MenuItem
											key={e.ID}
											value={String(e.ID)}
										>
											{e.Name}
										</MenuItem>
									))}
								</Select>
							</FormControl>
							<Button
								variant="contained"
								color="error"
								onClick={addEnemyToEncounter}
								sx={{
									fontWeight: 600,
									fontSize: "1rem",
									px: 2,
									py: 1,
								}}
							>
								Add Enemy
							</Button>
						</Stack>
					</Stack>
				</CardContent>
			</Card>
		</Box>
	);
};
