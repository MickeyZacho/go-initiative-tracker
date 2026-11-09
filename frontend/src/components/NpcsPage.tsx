// Renamed from MonstersPage.tsx
import { useCallback, useEffect, useState } from "react";
import {
	Box,
	Button,
	Collapse,
	IconButton,
	MenuItem,
	Paper,
	Stack,
	TextField,
	Tooltip,
	Typography,
} from "@mui/material";
import { apiGetArray, apiPost } from "../lib/http";
import { useMe } from "../hooks/useMe";
import AddIcon from "@mui/icons-material/Add";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutline";
import PersonAddAlt1OutlinedIcon from "@mui/icons-material/PersonAddAlt1Outlined";
import { headingFont } from "../theme";
import { InlineInput } from "./InlineInput";

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
	OwnerID?: string;
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
	const [showCreate, setShowCreate] = useState(false);

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
			// Default the target to the first encounter so "Add to encounter"
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

	// Edit a template field in place (local only); persisted on blur.
	const updateNpcLocal = (id: number, patch: Partial<NpcTemplate>) => {
		setNpcs((prev) =>
			prev.map((n) => (n.ID === id ? { ...n, ...patch } : n)),
		);
	};

	// Auto-save on blur, matching the Characters page. Reload only on failure,
	// to put the server's values back.
	const handleBlurSave = async (npc: NpcTemplate) => {
		try {
			await saveNpc(npc);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to update npc",
			);
			await loadNpcs();
		}
	};

	// Shared (seed) templates and other users' templates are read-only; the
	// backend only updates rows whose owner matches the caller.
	const me = useMe();
	const canEdit = (npc: NpcTemplate) =>
		(npc.OwnerID ?? "") === (me.loggedIn ? me.discordID : "");

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

	const targetName =
		encounters.find((e) => e.ID === targetEncounterId)?.Name ?? "";

	return (
		<Stack spacing={2}>
			<Stack
				direction="row"
				alignItems="center"
				spacing={2}
				useFlexGap
				flexWrap="wrap"
			>
				<Typography variant="h5" sx={{ flexGrow: 1 }}>
					NPC Templates
				</Typography>
				{/* Where the per-card "add" button sends a template. */}
				<TextField
					select
					size="small"
					label="Add to encounter"
					value={encounters.length ? targetEncounterId : ""}
					onChange={(e) => setTargetEncounterId(Number(e.target.value))}
					disabled={encounters.length === 0}
					sx={{ minWidth: 200 }}
				>
					{encounters.map((encounter) => (
						<MenuItem key={encounter.ID} value={encounter.ID}>
							{encounter.Name}
						</MenuItem>
					))}
				</TextField>
				<Button
					variant={showCreate ? "outlined" : "contained"}
					startIcon={<AddIcon />}
					onClick={() => setShowCreate((v) => !v)}
				>
					New NPC
				</Button>
			</Stack>
			{error && <Typography color="error">{error}</Typography>}
			{notice && <Typography color="success.main">{notice}</Typography>}
			{encounters.length === 0 && (
				<Typography variant="body2" color="text.secondary">
					Create an encounter first to add NPCs to it.
				</Typography>
			)}
			<Collapse in={showCreate} unmountOnExit>
				<Paper variant="outlined" sx={{ p: 2 }}>
					<Box
						sx={{
							display: "grid",
							gridTemplateColumns: {
								xs: "repeat(2, 1fr)",
								sm: "repeat(4, 1fr)",
								md: "repeat(8, 1fr)",
							},
							gap: 2,
							alignItems: "start",
						}}
					>
						<TextField
							label="Name"
							size="small"
							autoFocus
							sx={{ gridColumn: { xs: "1 / -1", sm: "span 2", md: "span 3" } }}
							value={draft.Name}
							onChange={(e) =>
								setDraft((prev) => ({ ...prev, Name: e.target.value }))
							}
						/>
						<TextField
							label="Description"
							size="small"
							sx={{ gridColumn: { xs: "1 / -1", sm: "span 2", md: "span 3" } }}
							value={draft.Description}
							onChange={(e) =>
								setDraft((prev) => ({
									...prev,
									Description: e.target.value,
								}))
							}
						/>
						<TextField
							label="AC"
							type="number"
							size="small"
							value={draft.ArmorClass}
							onChange={(e) =>
								setDraft((prev) => ({
									...prev,
									ArmorClass: Number(e.target.value),
								}))
							}
						/>
						<TextField
							label="Max HP"
							type="number"
							size="small"
							value={draft.MaxHP}
							onChange={(e) =>
								setDraft((prev) => ({
									...prev,
									MaxHP: Number(e.target.value),
								}))
							}
						/>
						{STAT_KEYS.map((stat) => (
							<TextField
								key={stat}
								label={STAT_ABBR[stat]}
								type="number"
								size="small"
								value={draft.BaseStats[stat]}
								onChange={(e) =>
									setDraft((prev) => ({
										...prev,
										BaseStats: {
											...prev.BaseStats,
											[stat]: Number(e.target.value),
										},
									}))
								}
							/>
						))}
						<Stack
							direction="row"
							spacing={1}
							sx={{ gridColumn: { xs: "1 / -1", md: "span 2" } }}
						>
							<Button variant="contained" onClick={handleCreate}>
								Create
							</Button>
							<Button onClick={() => setShowCreate(false)}>Cancel</Button>
						</Stack>
					</Box>
				</Paper>
			</Collapse>
			{/* Stat-block cards. Your own templates edit in place and save when a
			    field loses focus; shared ones are read-only. */}
			<Box
				sx={{
					display: "grid",
					gridTemplateColumns: { xs: "1fr", md: "repeat(2, 1fr)" },
					gap: 1.5,
				}}
			>
				{npcs.map((npc) => {
					const editable = canEdit(npc);
					const save = () => handleBlurSave(npc);
					// A stat value: inline-editable for your templates, text otherwise.
					const value = (
						v: number,
						label: string,
						onChange: (v: number) => void,
					) =>
						editable ? (
							<InlineInput
								type="number"
								value={v}
								onChange={(e) => onChange(Number(e.target.value))}
								onBlur={save}
								inputProps={{
									"aria-label": `${npc.Name} ${label}`,
									style: { textAlign: "center" },
								}}
								sx={{ width: 56, fontWeight: 600 }}
							/>
						) : (
							<Typography fontWeight={600} sx={{ py: 0.5 }}>
								{v}
							</Typography>
						);
					return (
						<Paper
							key={npc.ID}
							variant="outlined"
							sx={{ p: 2, display: "flex", flexDirection: "column", gap: 1.5 }}
						>
							<Stack direction="row" alignItems="flex-start" spacing={1}>
								<Box sx={{ flex: 1, minWidth: 0 }}>
									{editable ? (
										<>
											<InlineInput
												value={npc.Name}
												onChange={(e) =>
													updateNpcLocal(npc.ID, { Name: e.target.value })
												}
												onBlur={save}
												inputProps={{ "aria-label": "Name" }}
												fullWidth
												sx={{
													fontFamily: headingFont,
													fontWeight: 700,
													fontSize: "1.1rem",
													color: "primary.main",
												}}
											/>
											<InlineInput
												value={npc.Description}
												placeholder="Description"
												onChange={(e) =>
													updateNpcLocal(npc.ID, {
														Description: e.target.value,
													})
												}
												onBlur={save}
												inputProps={{ "aria-label": "Description" }}
												fullWidth
												sx={{
													fontSize: "0.875rem",
													fontStyle: "italic",
													color: "text.secondary",
												}}
											/>
										</>
									) : (
										<Box sx={{ px: 0.75 }}>
											<Typography
												sx={{
													fontFamily: headingFont,
													fontWeight: 700,
													fontSize: "1.1rem",
													color: "primary.main",
												}}
											>
												{npc.Name}
											</Typography>
											<Typography
												variant="body2"
												color="text.secondary"
												sx={{ fontStyle: "italic" }}
											>
												{npc.Description || "Shared template"}
											</Typography>
										</Box>
									)}
								</Box>
								<Tooltip
									title={
										targetName ? `Add to ${targetName}` : "Pick an encounter first"
									}
								>
									<span>
										<IconButton
											aria-label={`Add ${npc.Name} to ${targetName || "encounter"}`}
											disabled={targetEncounterId <= 0}
											onClick={() => createCharacterFromTemplate(npc)}
											sx={{ color: "primary.main" }}
										>
											<PersonAddAlt1OutlinedIcon fontSize="small" />
										</IconButton>
									</span>
								</Tooltip>
								{editable && (
									<Tooltip title="Delete template">
										<IconButton
											aria-label={`Delete ${npc.Name}`}
											onClick={() => handleDelete(npc.ID)}
											sx={{
												color: "text.disabled",
												"&:hover": { color: "error.main" },
											}}
										>
											<DeleteOutlineIcon fontSize="small" />
										</IconButton>
									</Tooltip>
								)}
							</Stack>
							{/* AC and HP on a ruled line, like a printed stat block. */}
							<Stack
								direction="row"
								spacing={3}
								alignItems="center"
								sx={{
									borderTop: 2,
									borderBottom: 1,
									borderColor: "divider",
									py: 0.5,
									px: 0.75,
								}}
							>
								<Stack direction="row" alignItems="center" spacing={0.5}>
									<Typography variant="caption" color="text.secondary" fontWeight={700}>
										AC
									</Typography>
									{value(npc.ArmorClass, "AC", (n) =>
										updateNpcLocal(npc.ID, { ArmorClass: n }),
									)}
								</Stack>
								<Stack direction="row" alignItems="center" spacing={0.5}>
									<Typography variant="caption" color="text.secondary" fontWeight={700}>
										HP
									</Typography>
									{value(npc.MaxHP, "max HP", (n) =>
										updateNpcLocal(npc.ID, { MaxHP: n }),
									)}
								</Stack>
							</Stack>
							{/* Ability scores with their modifiers. */}
							<Box
								sx={{
									display: "grid",
									gridTemplateColumns: "repeat(6, minmax(0, 1fr))",
								}}
							>
								{STAT_KEYS.map((stat) => {
									const score = npc.BaseStats[stat];
									const mod = Math.floor((score - 10) / 2);
									return (
										<Box
											key={stat}
											sx={{
												display: "flex",
												flexDirection: "column",
												alignItems: "center",
											}}
										>
											<Typography variant="caption" color="text.secondary" fontWeight={700}>
												{STAT_ABBR[stat]}
											</Typography>
											{value(score, STAT_ABBR[stat], (n) =>
												updateNpcLocal(npc.ID, {
													BaseStats: { ...npc.BaseStats, [stat]: n },
												}),
											)}
											<Typography variant="caption" color="text.secondary">
												{mod >= 0 ? `+${mod}` : mod}
											</Typography>
										</Box>
									);
								})}
							</Box>
						</Paper>
					);
				})}
			</Box>
		</Stack>
	);
}
