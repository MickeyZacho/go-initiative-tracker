import { Fragment, useCallback, useEffect, useState } from "react";
import {
	Avatar,
	Box,
	Button,
	Chip,
	Divider,
	MenuItem,
	Paper,
	Select,
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
import { useMe } from "../hooks/useMe";
import { useFriends } from "../hooks/useFriends";
import { useEncounterMembers } from "../hooks/useEncounterMembers";
import { discordAvatarUrl } from "../lib/discord";

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
	const [sharingId, setSharingId] = useState<number | null>(null);
	const [selectedFriend, setSelectedFriend] = useState("");

	const me = useMe();
	const { friends } = useFriends();
	const { members, error: membersError, load, addMember, removeMember, clear } =
		useEncounterMembers();

	const loadEncounters = useCallback(async () => {
		setLoading(true);
		setError("");
		try {
			setEncounters(await apiGetArray<Encounter>("/encounters"));
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
			await apiPost(
				"/encounters/save",
				{ Name: name.trim(), Description: description.trim() },
				"Failed to create encounter",
			);
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
			await apiPost(
				"/encounters/delete",
				{ id },
				"Failed to delete encounter",
			);
			await loadEncounters();
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to delete encounter",
			);
		}
	};

	const toggleSharing = (encounterId: number) => {
		if (sharingId === encounterId) {
			setSharingId(null);
			clear();
			return;
		}
		setSharingId(encounterId);
		setSelectedFriend("");
		load(encounterId);
	};

	const handleAddMember = async () => {
		if (sharingId === null || !selectedFriend) return;
		setError("");
		try {
			await addMember(sharingId, selectedFriend);
			setSelectedFriend("");
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to add member",
			);
		}
	};

	const handleRemoveMember = async (userId: string) => {
		if (sharingId === null) return;
		try {
			await removeMember(sharingId, userId);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to remove member",
			);
		}
	};

	// Friends not already sharing this encounter, available to add.
	const memberIds = new Set(members.map((m) => m.discord_id));
	const addableFriends = friends.filter((f) => !memberIds.has(f.discord_id));

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
						{encounters.map((encounter) => {
							const isOwner =
								me.loggedIn &&
								encounter.OwnerID === me.discordID;
							return (
								<Fragment key={encounter.ID}>
									<TableRow>
										<TableCell>
											{encounter.Name}
											{!isOwner && me.loggedIn && (
												<Chip
													label="Shared with you"
													size="small"
													sx={{ ml: 1 }}
												/>
											)}
										</TableCell>
										<TableCell>
											{encounter.Description}
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
														onOpenEncounter(
															encounter.ID,
														)
													}
												>
													Open
												</Button>
												{isOwner && (
													<Button
														size="small"
														variant="outlined"
														onClick={() =>
															toggleSharing(
																encounter.ID,
															)
														}
													>
														Share
													</Button>
												)}
												{isOwner && (
													<Button
														size="small"
														color="error"
														variant="outlined"
														onClick={() =>
															handleDelete(
																encounter.ID,
															)
														}
													>
														Delete
													</Button>
												)}
											</Stack>
										</TableCell>
									</TableRow>
									{sharingId === encounter.ID && (
										<TableRow>
											<TableCell colSpan={3}>
												<Box sx={{ p: 1 }}>
													<Typography
														variant="subtitle2"
														fontWeight={700}
														gutterBottom
													>
														Shared with
													</Typography>
													{membersError && (
														<Typography color="error">
															{membersError}
														</Typography>
													)}
													<Stack
														spacing={1}
														divider={
															<Divider flexItem />
														}
														mb={2}
													>
														{members.length ===
														0 ? (
															<Typography color="text.secondary">
																Not shared with
																anyone yet.
															</Typography>
														) : (
															members.map((m) => (
																<Stack
																	key={
																		m.discord_id
																	}
																	direction="row"
																	spacing={2}
																	alignItems="center"
																>
																	<Avatar
																		src={
																			discordAvatarUrl(
																				m.discord_id,
																				m.avatar,
																			) ??
																			undefined
																		}
																		sx={{
																			width: 28,
																			height: 28,
																		}}
																	>
																		{m.username
																			.charAt(
																				0,
																			)
																			.toUpperCase()}
																	</Avatar>
																	<Typography
																		flexGrow={
																			1
																		}
																	>
																		{
																			m.username
																		}
																	</Typography>
																	<Button
																		size="small"
																		color="error"
																		variant="outlined"
																		onClick={() =>
																			handleRemoveMember(
																				m.discord_id,
																			)
																		}
																	>
																		Remove
																	</Button>
																</Stack>
															))
														)}
													</Stack>
													<Stack
														direction="row"
														spacing={2}
														alignItems="center"
													>
														<Select
															size="small"
															displayEmpty
															value={
																selectedFriend
															}
															onChange={(e) =>
																setSelectedFriend(
																	e.target
																		.value,
																)
															}
															sx={{ minWidth: 200 }}
														>
															<MenuItem value="">
																{addableFriends.length ===
																0
																	? "No friends to add"
																	: "Select a friend…"}
															</MenuItem>
															{addableFriends.map(
																(f) => (
																	<MenuItem
																		key={
																			f.discord_id
																		}
																		value={
																			f.discord_id
																		}
																	>
																		{
																			f.username
																		}
																	</MenuItem>
																),
															)}
														</Select>
														<Button
															variant="contained"
															size="small"
															disabled={
																!selectedFriend
															}
															onClick={
																handleAddMember
															}
														>
															Add
														</Button>
													</Stack>
												</Box>
											</TableCell>
										</TableRow>
									)}
								</Fragment>
							);
						})}
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
