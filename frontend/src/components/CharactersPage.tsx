import { useCallback, useEffect, useState } from "react";
import {
	Box,
	Button,
	MenuItem,
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
import { apiGetArray, apiPost } from "../lib/http";

interface Character {
	ID: number;
	Name: string;
	ArmorClass: number;
	ToHitModifier?: number;
	MaxHP: number;
	OwnerID: string;
	Type: string; // 'pc' or 'npc'
	NpcTemplateID?: number; // for NPCs, reference to their template
}

const emptyCharacter: Character = {
	ID: 0,
	Name: "",
	ArmorClass: 10,
	ToHitModifier: 0,
	MaxHP: 10,
	OwnerID: "",
	Type: "pc",
	NpcTemplateID: undefined,
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
			setCharacters(await apiGetArray<Character>("/characters/library"));
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

	// Persist a character to the backend. Returns true on success. Does not
	// reload the list — callers that change grouping (e.g. Type) reload after.
	const persist = useCallback(async (character: Character) => {
		setError("");
		if (!character.Name.trim()) {
			setError("Character name is required");
			return false;
		}
		if (character.MaxHP < 1) {
			setError("Max HP must be at least 1");
			return false;
		}
		try {
			await apiPost(
				"/characters/library/save",
				character,
				"Failed to save character",
			);
			return true;
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to save character",
			);
			return false;
		}
	}, []);

	const handleCreate = async () => {
		if (await persist(draft)) {
			setDraft(emptyCharacter);
			await loadCharacters();
		}
	};

	// Update a single field in local state (does not save).
	const updateField = <K extends keyof Character>(
		id: number,
		field: K,
		value: Character[K],
	) => {
		setCharacters((prev) =>
			prev.map((c) => (c.ID === id ? { ...c, [field]: value } : c)),
		);
	};

	// Auto-save on blur — the row's current state is already up to date.
	const handleBlur = (character: Character) => {
		void persist(character);
	};

	// Changing Type re-groups the row, so save then reload.
	const handleTypeChange = async (character: Character, type: string) => {
		const updated = { ...character, Type: type };
		updateField(character.ID, "Type", type);
		if (await persist(updated)) {
			await loadCharacters();
		}
	};

	const handleDelete = async (id: number) => {
		if (!window.confirm("Delete this character?")) {
			return;
		}
		try {
			await apiPost(
				"/characters/library/delete",
				{ id },
				"Failed to delete character",
			);
			await loadCharacters();
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to delete character",
			);
		}
	};

	const renderRow = (character: Character) => (
		<TableRow key={character.ID}>
			<TableCell>
				<TextField
					variant="standard"
					value={character.Name}
					onChange={(e) =>
						updateField(character.ID, "Name", e.target.value)
					}
					onBlur={() => handleBlur(character)}
				/>
			</TableCell>
			<TableCell>
				<TextField
					variant="standard"
					type="number"
					value={character.ArmorClass}
					onChange={(e) =>
						updateField(
							character.ID,
							"ArmorClass",
							Number(e.target.value),
						)
					}
					onBlur={() => handleBlur(character)}
				/>
			</TableCell>
			<TableCell>
				<TextField
					variant="standard"
					type="number"
					value={character.ToHitModifier ?? 0}
					onChange={(e) =>
						updateField(
							character.ID,
							"ToHitModifier",
							Number(e.target.value),
						)
					}
					onBlur={() => handleBlur(character)}
				/>
			</TableCell>
			<TableCell>
				<TextField
					variant="standard"
					type="number"
					value={character.MaxHP}
					onChange={(e) =>
						updateField(
							character.ID,
							"MaxHP",
							Number(e.target.value),
						)
					}
					onBlur={() => handleBlur(character)}
				/>
			</TableCell>
			<TableCell>
				<TextField
					select
					variant="standard"
					value={character.Type}
					onChange={(e) =>
						handleTypeChange(character, e.target.value)
					}
				>
					<MenuItem value="pc">PC</MenuItem>
					<MenuItem value="npc">NPC</MenuItem>
				</TextField>
			</TableCell>
			<TableCell align="right">
				<Button
					size="small"
					color="error"
					variant="outlined"
					onClick={() => handleDelete(character.ID)}
				>
					Delete
				</Button>
			</TableCell>
		</TableRow>
	);

	const pcs = characters.filter((c) => c.Type !== "npc");
	const npcs = characters.filter((c) => c.Type === "npc");

	return (
		<Stack spacing={2}>
			<Typography variant="h5" fontWeight={700}>
				Characters
			</Typography>
			{error && <Typography color="error">{error}</Typography>}
			<Paper sx={{ p: 2 }}>
				<Stack
					direction={{ xs: "column", sm: "row" }}
					spacing={2}
					alignItems={{ xs: "stretch", sm: "center" }}
				>
					<TextField
						label="Name"
						size="small"
						sx={{ flex: 1, minWidth: 160 }}
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
						size="small"
						sx={{ width: { xs: "100%", sm: 90 } }}
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
						size="small"
						sx={{ width: { xs: "100%", sm: 90 } }}
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
						size="small"
						sx={{ width: { xs: "100%", sm: 90 } }}
						value={draft.MaxHP}
						onChange={(e) =>
							setDraft((prev) => ({
								...prev,
								MaxHP: Number(e.target.value),
							}))
						}
					/>
					<Button
						variant="contained"
						onClick={handleCreate}
						sx={{ flexShrink: 0 }}
					>
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
							<TableCell>Type</TableCell>
							<TableCell align="right">Actions</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{pcs.length > 0 && (
							<TableRow>
								<TableCell
									colSpan={6}
									style={{
										background: "#e3f2fd",
										fontWeight: 700,
									}}
								>
									Player Characters (PCs)
								</TableCell>
							</TableRow>
						)}
						{pcs.map(renderRow)}
						{npcs.length > 0 && (
							<TableRow>
								<TableCell
									colSpan={6}
									style={{
										background: "#fce4ec",
										fontWeight: 700,
									}}
								>
									Non-Player Characters (NPCs)
								</TableCell>
							</TableRow>
						)}
						{npcs.map(renderRow)}
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
