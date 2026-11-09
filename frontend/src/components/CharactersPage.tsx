import { useCallback, useEffect, useState } from "react";
import {
	Box,
	Button,
	IconButton,
	MenuItem,
	Paper,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableRow,
	TextField,
	Tooltip,
	Typography,
} from "@mui/material";
import { apiGetArray, apiPost } from "../lib/http";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutline";
import { InlineInput } from "./InlineInput";

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

	// Number fields share one shape: right-aligned digits, narrow column.
	const numberField = (
		character: Character,
		field: "ArmorClass" | "ToHitModifier" | "MaxHP",
		label: string,
	) => (
		<InlineInput
			type="number"
			value={character[field] ?? 0}
			onChange={(e) =>
				updateField(character.ID, field, Number(e.target.value))
			}
			onBlur={() => handleBlur(character)}
			inputProps={{ "aria-label": `${character.Name} ${label}` }}
			sx={{ width: 72 }}
		/>
	);

	const renderRow = (character: Character) => (
		<TableRow key={character.ID} hover>
			<TableCell>
				<InlineInput
					value={character.Name}
					onChange={(e) =>
						updateField(character.ID, "Name", e.target.value)
					}
					onBlur={() => handleBlur(character)}
					inputProps={{ "aria-label": "Name" }}
					sx={{ fontWeight: 600, width: "100%", minWidth: 120 }}
				/>
			</TableCell>
			<TableCell>{numberField(character, "ArmorClass", "AC")}</TableCell>
			<TableCell>{numberField(character, "ToHitModifier", "to-hit")}</TableCell>
			<TableCell>{numberField(character, "MaxHP", "max HP")}</TableCell>
			<TableCell>
				<TextField
					select
					variant="standard"
					value={character.Type}
					onChange={(e) => handleTypeChange(character, e.target.value)}
					slotProps={{
						input: { disableUnderline: true },
						htmlInput: { "aria-label": `${character.Name} type` },
					}}
					sx={{ minWidth: 64 }}
				>
					<MenuItem value="pc">PC</MenuItem>
					<MenuItem value="npc">NPC</MenuItem>
				</TextField>
			</TableCell>
			<TableCell align="right">
				<Tooltip title="Delete character">
					<IconButton
						size="small"
						aria-label={`Delete ${character.Name}`}
						onClick={() => handleDelete(character.ID)}
						sx={{
							color: "text.disabled",
							"&:hover": { color: "error.main" },
						}}
					>
						<DeleteOutlineIcon fontSize="small" />
					</IconButton>
				</Tooltip>
			</TableCell>
		</TableRow>
	);

	// Section label inside the table (PCs / NPCs).
	const groupRow = (label: string) => (
		<TableRow>
			<TableCell
				colSpan={6}
				sx={{
					pt: 2,
					color: "text.secondary",
					fontSize: "0.75rem",
					fontWeight: 700,
					textTransform: "uppercase",
					letterSpacing: "0.08em",
				}}
			>
				{label}
			</TableCell>
		</TableRow>
	);

	const pcs = characters.filter((c) => c.Type !== "npc");
	const npcs = characters.filter((c) => c.Type === "npc");

	return (
		<Stack spacing={2}>
			<Typography variant="h5">Characters</Typography>
			{error && <Typography color="error">{error}</Typography>}
			<Paper variant="outlined" sx={{ p: 2 }}>
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
							setDraft((prev) => ({ ...prev, Name: e.target.value }))
						}
					/>
					<TextField
						label="AC"
						type="number"
						size="small"
						sx={{ width: { xs: "100%", sm: 80 } }}
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
						sx={{ width: { xs: "100%", sm: 80 } }}
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
						sx={{ width: { xs: "100%", sm: 80 } }}
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
						New character
					</Button>
				</Stack>
			</Paper>
			<Paper variant="outlined" sx={{ overflowX: "auto" }}>
				<Table size="small">
					<TableHead>
						<TableRow>
							<TableCell>Name</TableCell>
							<TableCell>AC</TableCell>
							<TableCell>To-Hit</TableCell>
							<TableCell>Max HP</TableCell>
							<TableCell>Type</TableCell>
							<TableCell align="right" />
						</TableRow>
					</TableHead>
					<TableBody>
						{pcs.length > 0 && groupRow("Player characters")}
						{pcs.map(renderRow)}
						{npcs.length > 0 && groupRow("NPCs")}
						{npcs.map(renderRow)}
					</TableBody>
				</Table>
				{!loading && characters.length === 0 && (
					<Box sx={{ p: 4, textAlign: "center" }}>
						<Typography fontWeight={600}>No characters yet</Typography>
						<Typography variant="body2" color="text.secondary">
							Add your party above; they'll be ready to drop into any
							encounter.
						</Typography>
					</Box>
				)}
			</Paper>
		</Stack>
	);
}
