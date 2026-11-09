// Renamed from MonstersPage.tsx
import { useCallback, useEffect, useState } from "react";
import {
	Box,
	Button,
	MenuItem,
	Paper,
	Stack,
	TextField,
	Typography,
} from "@mui/material";
import { apiGetArray, apiPost } from "../lib/http";

interface StatBlock {
	Strength: number;
	Dexterity: number;
	Constitution: number;
	Intelligence: number;
	Wisdom: number;
	Charisma: number;
}

interface NpcTemplate {
	ID: number;
	Name: string;
	Description: string;
	BaseStats: StatBlock;
	ArmorClass: number;
	MaxHP: number;
}

interface EncounterOption {
	ID: number;
	Name: string;
}

const STAT_KEYS: (keyof StatBlock)[] = [
	"Strength",
	"Dexterity",
	"Constitution",
	"Intelligence",
	"Wisdom",
	"Charisma",
];
const STAT_ABBR: Record<keyof StatBlock, string> = {
	Strength: "STR",
	Dexterity: "DEX",
	Constitution: "CON",
	Intelligence: "INT",
	Wisdom: "WIS",
	Charisma: "CHA",
};

const emptyStatBlock: StatBlock = {
	Strength: 10,
	Dexterity: 10,
	Constitution: 10,
	Intelligence: 10,
	Wisdom: 10,
	Charisma: 10,
};

const emptyNpc: NpcTemplate = {
	ID: 0,
	Name: "",
	Description: "",
	BaseStats: { ...emptyStatBlock },
	ArmorClass: 10,
	MaxHP: 10,
};

export default function NpcsPage() {
	const [npcs, setNpcs] = useState<NpcTemplate[]>([]);
	const [draft, setDraft] = useState<NpcTemplate>(emptyNpc);
	const [error, setError] = useState("");
	const [notice, setNotice] = useState("");
	const [encounters, setEncounters] = useState<EncounterOption[]>([]);
	const [targetEncounterId, setTargetEncounterId] = useState<number>(0);

	const loadNpcs = useCallback(async () => {
		setError("");
		try {
			setNpcs(await apiGetArray<NpcTemplate>("/npcs/templates"));
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to load npcs",
			);
			setNpcs([]);
		}
	}, []);

	const loadEncounters = useCallback(async () => {
		try {
			const data = await apiGetArray<EncounterOption>("/encounters");
			setEncounters(data);
			// Default the target to the first encounter so "Create Character"
			// works without an extra click when one exists.
			setTargetEncounterId((current) =>
				current > 0 ? current : (data[0]?.ID ?? 0),
			);
		} catch {
			setEncounters([]);
		}
	}, []);

	useEffect(() => {
		loadNpcs();
		loadEncounters();
	}, [loadNpcs, loadEncounters]);

	const saveNpc = async (npc: NpcTemplate) => {
		setError("");
		if (!npc.Name.trim()) {
			setError("NPC name is required");
			return;
		}
		await apiPost("/npcs/templates/save", npc, "Failed to save npc template");
	};

	const createCharacterFromTemplate = async (template: NpcTemplate) => {
		setError("");
		setNotice("");
		if (targetEncounterId <= 0) {
			setError("Select a target encounter first");
			return;
		}
		try {
			// The backend expects npc_template_id + encounter_id (both > 0).
			await apiPost(
				"/npcs/templates/create-character",
				{
					npc_template_id: template.ID,
					encounter_id: targetEncounterId,
				},
				"Failed to create character from template",
			);
			const encounterName =
				encounters.find((e) => e.ID === targetEncounterId)?.Name ??
				"the encounter";
			setNotice(`Added ${template.Name} to ${encounterName}.`);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to create character from template",
			);
		}
	};

	const handleCreate = async () => {
		try {
			await saveNpc(draft);
			setDraft(emptyNpc);
			await loadNpcs();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to create npc",
			);
		}
	};

	const handleUpdate = async (npc: NpcTemplate) => {
		try {
			await saveNpc(npc);
			await loadNpcs();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to update npc",
			);
		}
	};

	const handleDelete = async (id: number) => {
		if (!window.confirm("Delete this npc template?")) {
			return;
		}
		try {
			await apiPost(
				"/npcs/templates/delete",
				{ id },
				"Failed to delete npc template",
			);
			await loadNpcs();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to delete npc",
			);
		}
	};

	return (
		<Stack spacing={2}>
			<Typography variant="h5" fontWeight={700}>
				NPC Templates
			</Typography>
			{error && <Typography color="error">{error}</Typography>}
			{notice && <Typography color="success.main">{notice}</Typography>}
			<Paper sx={{ p: 2 }}>
				<Box
					sx={{
						display: "grid",
						gridTemplateColumns: {
							xs: "repeat(2, 1fr)",
							sm: "repeat(3, 1fr)",
							md: "repeat(4, 1fr)",
						},
						gap: 2,
						alignItems: "start",
					}}
				>
					<TextField
						label="Name"
						size="small"
						sx={{ gridColumn: { xs: "1 / -1", sm: "span 2" } }}
						value={draft.Name}
						onChange={(e) =>
							setDraft((prev) => ({
								...prev,
								Name: e.target.value,
							}))
						}
					/>
					<TextField
						label="Description"
						size="small"
						sx={{ gridColumn: { xs: "1 / -1", sm: "span 2" } }}
						value={draft.Description}
						onChange={(e) =>
							setDraft((prev) => ({
								...prev,
								Description: e.target.value,
							}))
						}
					/>
					<TextField
						label="Armor Class"
						type="number"
						value={draft.ArmorClass}
						onChange={(e) =>
							setDraft((prev) => ({
								...prev,
								ArmorClass: Number(e.target.value),
							}))
						}
						size="small"
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
						size="small"
					/>
					{Object.entries(draft.BaseStats).map(([stat, value]) => (
						<TextField
							key={stat}
							label={stat.charAt(0).toUpperCase() + stat.slice(1)}
							type="number"
							value={value}
							onChange={(e) =>
								setDraft((prev) => ({
									...prev,
									BaseStats: {
										...prev.BaseStats,
										[stat]: Number(e.target.value),
									},
								}))
							}
							size="small"
						/>
					))}
					<Button
						variant="contained"
						onClick={handleCreate}
						sx={{ gridColumn: "1 / -1", justifySelf: "start" }}
					>
						New NPC
					</Button>
				</Box>
			</Paper>
			<Paper sx={{ p: 2 }}>
				<Stack direction="row" spacing={2} alignItems="center">
					<TextField
						select
						label="Add to encounter"
						value={encounters.length ? targetEncounterId : ""}
						onChange={(e) =>
							setTargetEncounterId(Number(e.target.value))
						}
						disabled={encounters.length === 0}
						helperText={
							encounters.length === 0
								? "Create an encounter first"
								: "Target for “Create Character”"
						}
						sx={{ minWidth: 220 }}
					>
						{encounters.map((encounter) => (
							<MenuItem key={encounter.ID} value={encounter.ID}>
								{encounter.Name}
							</MenuItem>
						))}
					</TextField>
				</Stack>
			</Paper>
			{/* Cards rather than an 11-column table, so the list fits a tablet or
			    phone without horizontal scrolling. */}
			<Stack spacing={1}>
				{npcs.map((npc) => (
					<Paper key={npc.ID} variant="outlined" sx={{ p: 1.5 }}>
						<Box
							sx={{
								display: "flex",
								flexWrap: "wrap",
								alignItems: "center",
								columnGap: 2,
								rowGap: 1,
							}}
						>
							<Box sx={{ flex: "1 1 200px", minWidth: 0 }}>
								<Typography fontWeight={600}>{npc.Name}</Typography>
								{npc.Description && (
									<Typography variant="body2" color="text.secondary">
										{npc.Description}
									</Typography>
								)}
							</Box>
							<Typography variant="body2" sx={{ whiteSpace: "nowrap" }}>
								<b>AC</b> {npc.ArmorClass} · <b>HP</b> {npc.MaxHP}
							</Typography>
							<Box sx={{ whiteSpace: "nowrap" }}>
								<Button size="small" onClick={() => handleUpdate(npc)}>
									Update
								</Button>
								<Button
									size="small"
									color="error"
									onClick={() => handleDelete(npc.ID)}
								>
									Delete
								</Button>
								<Button
									size="small"
									disabled={targetEncounterId <= 0}
									onClick={() => createCharacterFromTemplate(npc)}
								>
									Create Character
								</Button>
							</Box>
						</Box>
						<Box
							sx={{
								display: "grid",
								gridTemplateColumns: "repeat(6, minmax(0, 1fr))",
								gap: 0.5,
								mt: 1,
								maxWidth: 420,
							}}
						>
							{STAT_KEYS.map((stat) => (
								<Box key={stat} sx={{ textAlign: "center" }}>
									<Typography variant="caption" color="text.secondary">
										{STAT_ABBR[stat]}
									</Typography>
									<Typography variant="body2">
										{npc.BaseStats[stat]}
									</Typography>
								</Box>
							))}
						</Box>
					</Paper>
				))}
			</Stack>
		</Stack>
	);
}
