// Renamed from MonstersPage.tsx
import { useCallback, useEffect, useState } from "react";
import {
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
						label="Description"
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
						sx={{ width: 100 }}
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
						sx={{ width: 100 }}
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
							sx={{ width: 100 }}
						/>
					))}
					<Button variant="contained" onClick={handleCreate}>
						New NPC
					</Button>
				</Stack>
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
			<Paper sx={{ p: 1 }}>
				<Table size="small">
					<TableHead>
						<TableRow>
							<TableCell>Name</TableCell>
							<TableCell>Description</TableCell>
							<TableCell>AC</TableCell>
							<TableCell>Max HP</TableCell>
							<TableCell>Strength</TableCell>
							<TableCell>Dexterity</TableCell>
							<TableCell>Constitution</TableCell>
							<TableCell>Intelligence</TableCell>
							<TableCell>Wisdom</TableCell>
							<TableCell>Charisma</TableCell>
							<TableCell align="right">Actions</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{npcs.map((npc) => (
							<TableRow key={npc.ID}>
								<TableCell>{npc.Name}</TableCell>
								<TableCell>{npc.Description}</TableCell>
								<TableCell>{npc.ArmorClass}</TableCell>
								<TableCell>{npc.MaxHP}</TableCell>
								<TableCell>{npc.BaseStats.Strength}</TableCell>
								<TableCell>{npc.BaseStats.Dexterity}</TableCell>
								<TableCell>
									{npc.BaseStats.Constitution}
								</TableCell>
								<TableCell>
									{npc.BaseStats.Intelligence}
								</TableCell>
								<TableCell>{npc.BaseStats.Wisdom}</TableCell>
								<TableCell>{npc.BaseStats.Charisma}</TableCell>
								<TableCell align="right">
									<Button
										size="small"
										onClick={() => handleUpdate(npc)}
									>
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
										onClick={() =>
											createCharacterFromTemplate(npc)
										}
									>
										Create Character
									</Button>
								</TableCell>
							</TableRow>
						))}
					</TableBody>
				</Table>
			</Paper>
		</Stack>
	);
}
