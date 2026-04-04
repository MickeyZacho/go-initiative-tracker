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

interface MonsterTemplate {
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

const emptyMonster: MonsterTemplate = {
	ID: 0,
	Name: "",
	Description: "",
	BaseStats: { ...emptyStatBlock },
	ArmorClass: 10,
	MaxHP: 10,
};

export default function MonstersPage() {
	const [monsters, setMonsters] = useState<MonsterTemplate[]>([]);
	const [draft, setDraft] = useState<MonsterTemplate>(emptyMonster);
	const [error, setError] = useState("");
	const [loading, setLoading] = useState(false);

	const loadMonsters = useCallback(async () => {
		setLoading(true);
		setError("");
		try {
			const response = await fetch(apiUrl("/api/monsters/templates"), {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to load monster templates");
			}
			const payload = await parseJsonResponse<unknown>(response);
			setMonsters(Array.isArray(payload) ? payload : []);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to load monsters",
			);
			setMonsters([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		loadMonsters();
	}, [loadMonsters]);

	const saveMonster = async (monster: MonsterTemplate) => {
		setError("");
		if (!monster.Name.trim()) {
			setError("Monster name is required");
			return;
		}
		const response = await fetch(apiUrl("/api/monsters/templates/save"), {
			method: "POST",
			credentials: "include",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(monster),
		});
		if (!response.ok) {
			const message = await response.text();
			throw new Error(message || "Failed to save monster template");
		}
	};

	const createCharacterFromTemplate = async (templateId: number) => {
		setError("");
		try {
			const response = await fetch(
				apiUrl("/api/monsters/templates/create-character"),
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
			await saveMonster(draft);
			setDraft(emptyMonster);
			await loadMonsters();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to create monster",
			);
		}
	};

	const handleUpdate = async (monster: MonsterTemplate) => {
		try {
			await saveMonster(monster);
			await loadMonsters();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to update monster",
			);
		}
	};

	const handleDelete = async (id: number) => {
		if (!window.confirm("Delete this monster template?")) {
			return;
		}
		try {
			const response = await fetch(
				apiUrl("/api/monsters/templates/delete"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ id }),
				},
			);
			if (!response.ok) {
				const message = await response.text();
				throw new Error(message || "Failed to delete monster template");
			}
			await loadMonsters();
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to delete monster",
			);
		}
	};

	return (
		<Stack spacing={2}>
			<Typography variant="h5" fontWeight={700}>
				Monster Templates
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
					<Button variant="contained" onClick={handleCreate}>
						New Monster
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
						{monsters.map((monster) => (
							<TableRow key={monster.ID}>
								<TableCell>
									<TextField
										variant="standard"
										value={monster.Name}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																name: e.target
																	.value,
															}
														: m,
												),
											)
										}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										value={monster.Description}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																Description:
																	e.target
																		.value,
															}
														: m,
												),
											)
										}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={monster.ArmorClass}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																ArmorClass:
																	Number(
																		e.target
																			.value,
																	),
															}
														: m,
												),
											)
										}
										sx={{ width: 80 }}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={monster.MaxHP}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																MaxHP: Number(
																	e.target
																		.value,
																),
															}
														: m,
												),
											)
										}
										sx={{ width: 80 }}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={monster.BaseStats?.Strength ?? 0}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																BaseStats: {
																	...m.BaseStats,
																	strength:
																		Number(
																			e
																				.target
																				.value,
																		),
																},
															}
														: m,
												),
											)
										}
										sx={{ width: 80 }}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={
											monster.BaseStats?.Dexterity ?? 0
										}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																BaseStats: {
																	...m.BaseStats,
																	Dexterity:
																		Number(
																			e
																				.target
																				.value,
																		),
																},
															}
														: m,
												),
											)
										}
										sx={{ width: 80 }}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={
											monster.BaseStats?.Constitution ?? 0
										}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																BaseStats: {
																	...m.BaseStats,
																	constitution:
																		Number(
																			e
																				.target
																				.value,
																		),
																},
															}
														: m,
												),
											)
										}
										sx={{ width: 80 }}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={
											monster.BaseStats?.Intelligence ?? 0
										}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																BaseStats: {
																	...m.BaseStats,
																	Intelligence:
																		Number(
																			e
																				.target
																				.value,
																		),
																},
															}
														: m,
												),
											)
										}
										sx={{ width: 80 }}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={monster.BaseStats?.Wisdom ?? 0}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																BaseStats: {
																	...m.BaseStats,
																	Wisdom: Number(
																		e.target
																			.value,
																	),
																},
															}
														: m,
												),
											)
										}
										sx={{ width: 80 }}
									/>
								</TableCell>
								<TableCell>
									<TextField
										variant="standard"
										type="number"
										value={monster.BaseStats?.Charisma ?? 0}
										onChange={(e) =>
											setMonsters((prev) =>
												prev.map((m) =>
													m.ID === monster.ID
														? {
																...m,
																BaseStats: {
																	...m.BaseStats,
																	Charisma:
																		Number(
																			e
																				.target
																				.value,
																		),
																},
															}
														: m,
												),
											)
										}
										sx={{ width: 80 }}
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
												handleUpdate(monster)
											}
										>
											Save
										</Button>
										<Button
											size="small"
											color="error"
											variant="outlined"
											onClick={() =>
												handleDelete(monster.ID)
											}
										>
											Delete
										</Button>
										<Button
											size="small"
											color="primary"
											variant="outlined"
											onClick={() =>
												createCharacterFromTemplate(
													monster.ID,
												)
											}
										>
											Add to Characters
										</Button>
									</Stack>
								</TableCell>
							</TableRow>
						))}
					</TableBody>
				</Table>
				{!loading && monsters.length === 0 && (
					<Box p={2}>
						<Typography color="text.secondary">
							No monster templates yet. Create your first one
							above.
						</Typography>
					</Box>
				)}
			</Paper>
		</Stack>
	);
}
