import { useCallback, useEffect, useState } from "react";
import {
	Box,
	Button,
	Paper,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableRow,
	TextField,
	Typography,
} from "@mui/material";
import { parseJsonResponse } from "../lib/http";
import { apiUrl } from "../lib/api";

interface Character {
	ID: number;
	Name: string;
	ArmorClass: number;
	ToHitModifier?: number;
	MaxHP: number;
	OwnerID: string;
}

const emptyCharacter: Character = {
	ID: 0,
	Name: "",
	ArmorClass: 10,
	ToHitModifier: 0,
	MaxHP: 10,
	OwnerID: "",
};

export default function CharactersPage() {
	const [characters, setCharacters] = useState<Character[]>([]);
	const [draft, setDraft] = useState<Character>(emptyCharacter);
	const [error, setError] = useState("");
	const [loading, setLoading] = useState(false);

	const loadCharacters = useCallback(async () => {
		setLoading(true);
		setError("");
		try {
			const response = await fetch(apiUrl("/api/characters/library"), {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to load characters");
			}
			const payload = await parseJsonResponse<unknown>(response);
			setCharacters(Array.isArray(payload) ? payload : []);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to load characters",
			);
			setCharacters([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		loadCharacters();
	}, [loadCharacters]);

	const saveCharacter = async (character: Character) => {
		setError("");
		if (!character.Name.trim()) {
			setError("Character name is required");
			return;
		}
		if (character.MaxHP < 1) {
			setError("Max HP must be at least 1");
			return;
		}

		const response = await fetch(apiUrl("/api/characters/library/save"), {
			method: "POST",
			credentials: "include",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(character),
		});

		if (!response.ok) {
			const message = await response.text();
			throw new Error(message || "Failed to save character");
		}
	};

	const handleCreate = async () => {
		try {
			await saveCharacter(draft);
			setDraft(emptyCharacter);
			await loadCharacters();
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to create character",
			);
		}
	};

	const handleUpdate = async (character: Character) => {
		try {
			await saveCharacter(character);
			await loadCharacters();
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to update character",
			);
		}
	};

	const handleDelete = async (id: number) => {
		if (!window.confirm("Delete this character?")) {
			return;
		}
		try {
			const response = await fetch(
				apiUrl("/api/characters/library/delete"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ id }),
				},
			);
			if (!response.ok) {
				const message = await response.text();
				throw new Error(message || "Failed to delete character");
			}
			await loadCharacters();
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to delete character",
			);
		}
	};

	return (
		<Stack spacing={2}>
			<Typography variant="h5" fontWeight={700}>
				Characters
			</Typography>
			{error && <Typography color="error">{error}</Typography>}
			<Paper sx={{ p: 2 }}>
				<Stack
					direction="row"
					spacing={2}
					alignItems="center"
					flexWrap="wrap"
				>
					<TextField
						label="Name"
						value={draft.Name}
						onChange={(e) =>
							setDraft((prev) => ({
								...prev,
								Name: e.target.value,
							}))
						}
					/>
					<TextField
						label="AC"
						type="number"
						value={draft.ArmorClass}
						onChange={(e) =>
							setDraft((prev) => ({
								...prev,
								ArmorClass: Number(e.target.value),
							}))
						}
					/>
					<TextField
						label="To-Hit"
						type="number"
						value={draft.ToHitModifier ?? 0}
						onChange={(e) =>
							setDraft((prev) => ({
								...prev,
								ToHitModifier: Number(e.target.value),
							}))
						}
					/>
					<TextField
						label="Max HP"
						type="number"
						value={draft.MaxHP}
						onChange={(e) =>
							setDraft((prev) => ({
								...prev,
								MaxHP: Number(e.target.value),
							}))
						}
					/>
					<Button variant="contained" onClick={handleCreate}>
						New Character
					</Button>
				</Stack>
			</Paper>
			<Paper sx={{ p: 1 }}>
				<Table size="small">
					<TableHead>
						<TableRow>
							<TableCell>Name</TableCell>
							<TableCell>AC</TableCell>
							<TableCell>To-Hit</TableCell>
							<TableCell>Max HP</TableCell>
							<TableCell align="right">Actions</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{characters.map((character) => (
							<TableRow key={character.ID}>
								<TableCell>
									<TextField
										variant="standard"
										value={character.Name}
										onChange={(e) =>
											setCharacters((prev) =>
												prev.map((c) =>
													c.ID === character.ID
														? {
																...c,
																Name: e.target
																	.value,
															}
														: c,
												),
											)
										}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={character.ArmorClass}
										onChange={(e) =>
											setCharacters((prev) =>
												prev.map((c) =>
													c.ID === character.ID
														? {
																...c,
																ArmorClass:
																	Number(
																		e.target
																			.value,
																	),
															}
														: c,
												),
											)
										}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={character.ToHitModifier ?? 0}
										onChange={(e) =>
											setCharacters((prev) =>
												prev.map((c) =>
													c.ID === character.ID
														? {
																...c,
																ToHitModifier:
																	Number(
																		e.target
																			.value,
																	),
															}
														: c,
												),
											)
										}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={character.MaxHP}
										onChange={(e) =>
											setCharacters((prev) =>
												prev.map((c) =>
													c.ID === character.ID
														? {
																...c,
																MaxHP: Number(
																	e.target
																		.value,
																),
															}
														: c,
												),
											)
										}
									/>
								</TableCell>
								<TableCell align="right">
									<Stack
										direction="row"
										spacing={1}
										justifyContent="flex-end"
									>
										<Button
											size="small"
											variant="contained"
											onClick={() =>
												handleUpdate(character)
											}
										>
											Save
										</Button>
										<Button
											size="small"
											color="error"
											variant="outlined"
											onClick={() =>
												handleDelete(character.ID)
											}
										>
											Delete
										</Button>
									</Stack>
								</TableCell>
							</TableRow>
						))}
					</TableBody>
				</Table>
				{!loading && characters.length === 0 && (
					<Box p={2}>
						<Typography color="text.secondary">
							No characters yet. Create your first one above.
						</Typography>
					</Box>
				)}
			</Paper>
		</Stack>
	);
}
