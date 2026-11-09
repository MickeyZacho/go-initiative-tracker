import { useCallback, useEffect, useState } from "react";
import {
	Avatar,
	Box,
	Button,
	Chip,
	Divider,
	IconButton,
	MenuItem,
	Paper,
	Select,
	Stack,
	TextField,
	Tooltip,
	Typography,
} from "@mui/material";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutline";
import GroupAddOutlinedIcon from "@mui/icons-material/GroupAddOutlined";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
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
			<Typography variant="h5">Encounters</Typography>
			{error && <Typography color="error">{error}</Typography>}
			<Paper variant="outlined" sx={{ p: 2 }}>
				<Stack
					direction={{ xs: "column", sm: "row" }}
					spacing={2}
					alignItems={{ xs: "stretch", sm: "center" }}
				>
					<TextField
						label="Encounter name"
						size="small"
						value={name}
						onChange={(e) => setName(e.target.value)}
						sx={{ flex: 1 }}
					/>
					<TextField
						label="Description"
						size="small"
						value={description}
						onChange={(e) => setDescription(e.target.value)}
						sx={{ flex: 2 }}
					/>
					<Button
						variant="contained"
						onClick={handleCreate}
						sx={{ flexShrink: 0 }}
					>
						New encounter
					</Button>
				</Stack>
			</Paper>
			<Stack spacing={1}>
				{encounters.map((encounter) => {
					const isOwner =
						me.loggedIn && encounter.OwnerID === me.discordID;
					return (
						<Paper
							key={encounter.ID}
							variant="outlined"
							sx={{
								p: 2,
								borderLeftWidth: 4,
								borderLeftColor: "primary.main",
							}}
						>
							<Stack
								direction="row"
								spacing={2}
								alignItems="center"
							>
								<Box sx={{ flex: 1, minWidth: 0 }}>
									<Stack
										direction="row"
										spacing={1}
										alignItems="center"
									>
										<Typography fontWeight={700} noWrap>
											{encounter.Name}
										</Typography>
										{!isOwner && me.loggedIn && (
											<Chip label="Shared with you" size="small" />
										)}
									</Stack>
									{encounter.Description && (
										<Typography
											variant="body2"
											color="text.secondary"
											sx={{ mt: 0.25 }}
										>
											{encounter.Description}
										</Typography>
									)}
								</Box>
								{isOwner && (
									<Tooltip title="Share with friends">
										<IconButton
											aria-label={`Share ${encounter.Name}`}
											aria-pressed={sharingId === encounter.ID}
											onClick={() => toggleSharing(encounter.ID)}
											sx={{
												color:
													sharingId === encounter.ID
														? "primary.main"
														: "text.secondary",
											}}
										>
											<GroupAddOutlinedIcon fontSize="small" />
										</IconButton>
									</Tooltip>
								)}
								{isOwner && (
									<Tooltip title="Delete encounter">
										<IconButton
											aria-label={`Delete ${encounter.Name}`}
											onClick={() => handleDelete(encounter.ID)}
											sx={{
												color: "text.disabled",
												"&:hover": { color: "error.main" },
											}}
										>
											<DeleteOutlineIcon fontSize="small" />
										</IconButton>
									</Tooltip>
								)}
								<Button
									variant="contained"
									startIcon={<PlayArrowIcon />}
									onClick={() => onOpenEncounter(encounter.ID)}
									sx={{ flexShrink: 0 }}
								>
									Open
								</Button>
							</Stack>
							{sharingId === encounter.ID && (
								<Box
									sx={{
										mt: 2,
										pt: 2,
										borderTop: 1,
										borderColor: "divider",
									}}
								>
									<Typography
										variant="overline"
										color="text.secondary"
										component="div"
									>
										Shared with
									</Typography>
									{membersError && (
										<Typography color="error">{membersError}</Typography>
									)}
									<Stack
										spacing={1}
										divider={<Divider flexItem />}
										mb={2}
									>
										{members.length === 0 ? (
											<Typography variant="body2" color="text.secondary">
												Not shared with anyone yet.
											</Typography>
										) : (
											members.map((m) => (
												<Stack
													key={m.discord_id}
													direction="row"
													spacing={2}
													alignItems="center"
												>
													<Avatar
														src={
															discordAvatarUrl(m.discord_id, m.avatar) ??
															undefined
														}
														sx={{ width: 28, height: 28 }}
													>
														{m.username.charAt(0).toUpperCase()}
													</Avatar>
													<Typography flexGrow={1}>{m.username}</Typography>
													<Button
														size="small"
														color="error"
														onClick={() => handleRemoveMember(m.discord_id)}
													>
														Remove
													</Button>
												</Stack>
											))
										)}
									</Stack>
									<Stack direction="row" spacing={2} alignItems="center">
										<Select
											size="small"
											displayEmpty
											value={selectedFriend}
											onChange={(e) => setSelectedFriend(e.target.value)}
											sx={{ minWidth: 200 }}
										>
											<MenuItem value="">
												{addableFriends.length === 0
													? "No friends to add"
													: "Select a friend…"}
											</MenuItem>
											{addableFriends.map((f) => (
												<MenuItem key={f.discord_id} value={f.discord_id}>
													{f.username}
												</MenuItem>
											))}
										</Select>
										<Button
											variant="contained"
											size="small"
											disabled={!selectedFriend}
											onClick={handleAddMember}
										>
											Add
										</Button>
									</Stack>
								</Box>
							)}
						</Paper>
					);
				})}
			</Stack>
			{!loading && encounters.length === 0 && (
				<Paper
					variant="outlined"
					sx={{ p: 4, textAlign: "center", borderStyle: "dashed" }}
				>
					<Typography fontWeight={600}>No encounters yet</Typography>
					<Typography variant="body2" color="text.secondary">
						Name one above, then open it to add combatants and roll
						initiative.
					</Typography>
				</Paper>
			)}
		</Stack>
	);
}
