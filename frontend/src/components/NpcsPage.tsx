// Renamed from MonstersPage.tsx
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
	const [loading, setLoading] = useState(false);

	const loadNpcs = useCallback(async () => {
		setLoading(true);
		setError("");
		try {
			const response = await fetch(apiUrl("/api/npcs/templates"), {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to load npc templates");
			}
			const payload = await parseJsonResponse<unknown>(response);
			setNpcs(Array.isArray(payload) ? payload : []);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to load npcs",
			);
			setNpcs([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		loadNpcs();
	}, [loadNpcs]);

	const saveNpc = async (npc: NpcTemplate) => {
		setError("");
		if (!npc.Name.trim()) {
			setError("NPC name is required");
			return;
		}
		const response = await fetch(apiUrl("/api/npcs/templates/save"), {
			method: "POST",
			credentials: "include",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(npc),
		});
		if (!response.ok) {
			const message = await response.text();
			throw new Error(message || "Failed to save npc template");
		}
	};

	const createCharacterFromTemplate = async (templateId: number) => {
		setError("");
		try {
			const response = await fetch(
				apiUrl("/api/npcs/templates/create-character"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ template_id: templateId }),
				},
			);
			if (!response.ok) {
				const message = await response.text();
				throw new Error(
					message || "Failed to create character from template",
				);
			}
			// Optionally, handle the created character here
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
			const response = await fetch(apiUrl("/api/npcs/templates/delete"), {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ id }),
			});
			if (!response.ok) {
				const message = await response.text();
				throw new Error(message || "Failed to delete npc template");
			}
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
										onClick={() =>
											createCharacterFromTemplate(npc.ID)
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
