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

interface Encounter {
	ID: number;
	Name: string;
	Description: string;
	OwnerID: string;
}

interface EncountersPageProps {
	onOpenEncounter: (encounterId: number) => void;
}

export default function EncountersPage({
	onOpenEncounter,
}: EncountersPageProps) {
	const [encounters, setEncounters] = useState<Encounter[]>([]);
	const [name, setName] = useState("");
	const [description, setDescription] = useState("");
	const [error, setError] = useState("");
	const [loading, setLoading] = useState(false);

	const loadEncounters = useCallback(async () => {
		setLoading(true);
		setError("");
		try {
			const response = await fetch("/api/encounters", {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to load encounters");
			}
			const payload = await parseJsonResponse<unknown>(response);
			setEncounters(
				Array.isArray(payload) ? (payload as Encounter[]) : [],
			);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to load encounters",
			);
			setEncounters([]);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		loadEncounters();
	}, [loadEncounters]);

	const handleCreate = async () => {
		setError("");
		if (!name.trim()) {
			setError("Encounter name is required");
			return;
		}
		try {
			const response = await fetch("/api/encounters/save", {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					Name: name.trim(),
					Description: description.trim(),
				}),
			});
			if (!response.ok) {
				const message = await response.text();
				throw new Error(message || "Failed to create encounter");
			}
			setName("");
			setDescription("");
			await loadEncounters();
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to create encounter",
			);
		}
	};

	const handleDelete = async (id: number) => {
		if (!window.confirm("Delete this encounter?")) {
			return;
		}
		try {
			const response = await fetch("/api/encounters/delete", {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ id }),
			});
			if (!response.ok) {
				const message = await response.text();
				throw new Error(message || "Failed to delete encounter");
			}
			await loadEncounters();
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to delete encounter",
			);
		}
	};

	return (
		<Stack spacing={2}>
			<Typography variant="h5" fontWeight={700}>
				Encounters
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
						label="Encounter Name"
						value={name}
						onChange={(e) => setName(e.target.value)}
					/>
					<TextField
						label="Description"
						value={description}
						onChange={(e) => setDescription(e.target.value)}
					/>
					<Button variant="contained" onClick={handleCreate}>
						New Encounter
					</Button>
				</Stack>
			</Paper>
			<Paper sx={{ p: 1 }}>
				<Table size="small">
					<TableHead>
						<TableRow>
							<TableCell>Name</TableCell>
							<TableCell>Description</TableCell>
							<TableCell align="right">Actions</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{encounters.map((encounter) => (
							<TableRow key={encounter.ID}>
								<TableCell>{encounter.Name}</TableCell>
								<TableCell>{encounter.Description}</TableCell>
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
												onOpenEncounter(encounter.ID)
											}
										>
											Open
										</Button>
										<Button
											size="small"
											color="error"
											variant="outlined"
											onClick={() =>
												handleDelete(encounter.ID)
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
				{!loading && encounters.length === 0 && (
					<Box p={2}>
						<Typography color="text.secondary">
							No encounters yet. Create your first one above.
						</Typography>
					</Box>
				)}
			</Paper>
		</Stack>
	);
}
