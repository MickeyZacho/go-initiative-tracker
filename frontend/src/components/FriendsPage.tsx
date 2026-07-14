import { useState } from "react";
import {
	Avatar,
	Box,
	Button,
	Divider,
	Paper,
	Stack,
	TextField,
	Typography,
} from "@mui/material";
import { useFriends, type Friend } from "../hooks/useFriends";
import { discordAvatarUrl } from "../lib/discord";

function FriendAvatar({ friend }: { friend: Friend }) {
	return (
		<Avatar
			src={discordAvatarUrl(friend.discord_id, friend.avatar) ?? undefined}
			sx={{ width: 32, height: 32 }}
		>
			{friend.username.charAt(0).toUpperCase()}
		</Avatar>
	);
}

export default function FriendsPage() {
	const {
		friends,
		incoming,
		outgoing,
		error,
		loading,
		sendRequest,
		accept,
		decline,
		remove,
	} = useFriends();
	const [username, setUsername] = useState("");
	const [actionError, setActionError] = useState("");
	const [notice, setNotice] = useState("");

	// Wrap a mutation so its thrown error (from the { status } envelope) surfaces
	// in the UI instead of an unhandled rejection.
	const run = async (fn: () => Promise<void>, successMsg?: string) => {
		setActionError("");
		setNotice("");
		try {
			await fn();
			if (successMsg) setNotice(successMsg);
		} catch (err) {
			setActionError(
				err instanceof Error ? err.message : "Something went wrong",
			);
		}
	};

	const handleAdd = async () => {
		if (!username.trim()) {
			setActionError("Enter a Discord username");
			return;
		}
		await run(async () => {
			await sendRequest(username);
			setUsername("");
		}, "Friend request sent");
	};

	return (
		<Stack spacing={2}>
			<Typography variant="h5" fontWeight={700}>
				Friends
			</Typography>
			<Typography color="text.secondary" variant="body2">
				Add friends by their Discord username. They must have signed in
				to this app at least once. Once you are friends, you can share
				encounters with them.
			</Typography>
			{error && <Typography color="error">{error}</Typography>}
			{actionError && <Typography color="error">{actionError}</Typography>}
			{notice && <Typography color="success.main">{notice}</Typography>}

			<Paper sx={{ p: 2 }}>
				<Stack direction="row" spacing={2} alignItems="center" flexWrap="wrap">
					<TextField
						label="Discord username"
						value={username}
						onChange={(e) => setUsername(e.target.value)}
						onKeyDown={(e) => {
							if (e.key === "Enter") handleAdd();
						}}
						size="small"
					/>
					<Button variant="contained" onClick={handleAdd}>
						Send Request
					</Button>
				</Stack>
			</Paper>

			{incoming.length > 0 && (
				<Paper sx={{ p: 2 }}>
					<Typography variant="subtitle1" fontWeight={700} gutterBottom>
						Incoming requests
					</Typography>
					<Stack spacing={1} divider={<Divider flexItem />}>
						{incoming.map((f) => (
							<Stack
								key={f.discord_id}
								direction="row"
								spacing={2}
								alignItems="center"
							>
								<FriendAvatar friend={f} />
								<Typography flexGrow={1}>{f.username}</Typography>
								<Button
									size="small"
									variant="contained"
									onClick={() =>
										run(() => accept(f.discord_id))
									}
								>
									Accept
								</Button>
								<Button
									size="small"
									color="error"
									variant="outlined"
									onClick={() =>
										run(() => decline(f.discord_id))
									}
								>
									Decline
								</Button>
							</Stack>
						))}
					</Stack>
				</Paper>
			)}

			{outgoing.length > 0 && (
				<Paper sx={{ p: 2 }}>
					<Typography variant="subtitle1" fontWeight={700} gutterBottom>
						Sent requests
					</Typography>
					<Stack spacing={1} divider={<Divider flexItem />}>
						{outgoing.map((f) => (
							<Stack
								key={f.discord_id}
								direction="row"
								spacing={2}
								alignItems="center"
							>
								<FriendAvatar friend={f} />
								<Typography flexGrow={1}>{f.username}</Typography>
								<Typography variant="body2" color="text.secondary">
									Pending
								</Typography>
								<Button
									size="small"
									color="error"
									variant="outlined"
									onClick={() =>
										run(() => remove(f.discord_id))
									}
								>
									Cancel
								</Button>
							</Stack>
						))}
					</Stack>
				</Paper>
			)}

			<Paper sx={{ p: 2 }}>
				<Typography variant="subtitle1" fontWeight={700} gutterBottom>
					Your friends
				</Typography>
				{friends.length === 0 ? (
					<Box p={1}>
						<Typography color="text.secondary">
							{loading ? "Loading…" : "No friends yet."}
						</Typography>
					</Box>
				) : (
					<Stack spacing={1} divider={<Divider flexItem />}>
						{friends.map((f) => (
							<Stack
								key={f.discord_id}
								direction="row"
								spacing={2}
								alignItems="center"
							>
								<FriendAvatar friend={f} />
								<Typography flexGrow={1}>{f.username}</Typography>
								<Button
									size="small"
									color="error"
									variant="outlined"
									onClick={() =>
										run(() => remove(f.discord_id))
									}
								>
									Remove
								</Button>
							</Stack>
						))}
					</Stack>
				)}
			</Paper>
		</Stack>
	);
}
