import React from "react";
import { Avatar, Box, Button } from "@mui/material";
import { apiGet } from "../lib/http";

const AuthButtons: React.FC = () => {
	const [username, setUsername] = React.useState<string>("");
	const [loggedIn, setLoggedIn] = React.useState<boolean>(false);
	const [discordID, setDiscordID] = React.useState<string>("");
	const [avatar, setAvatar] = React.useState<string>("");

	React.useEffect(() => {
		let mounted = true;
		apiGet<{
			loggedIn?: boolean;
			username?: string;
			discordID?: string;
			avatar?: string;
		}>("/me")
			.then((data) => {
				if (!mounted) return;
				setLoggedIn(Boolean(data.loggedIn));
				setUsername(data.username || "");
				setDiscordID(data.discordID || "");
				setAvatar(data.avatar || "");
			})
			.catch(() => {
				if (!mounted) return;
				setLoggedIn(false);
				setUsername("");
				setDiscordID("");
				setAvatar("");
			});
		return () => {
			mounted = false;
		};
	}, []);

	const avatarUrl =
		discordID && avatar
			? `https://cdn.discordapp.com/avatars/${discordID}/${avatar}.png`
			: undefined;

	if (!loggedIn) {
		return (
			<Button
				size="small"
				variant="contained"
				onClick={() => {
					window.location.href = "/api/login/discord";
				}}
			>
				Login with Discord
			</Button>
		);
	}

	// Logged in: identity is conveyed by the avatar alone, which stays fixed in
	// place. The logout button is positioned absolutely to the right of the
	// avatar and slides in on hover/focus. It's a descendant of the container, so
	// hovering the button keeps the reveal open even though it sits outside the
	// avatar's box.
	return (
		<Box
			sx={{
				position: "relative",
				display: "inline-flex",
				alignItems: "center",
				flexShrink: 0,
				"&:hover .logout-reveal, &:focus-within .logout-reveal": {
					opacity: 1,
					pointerEvents: "auto",
					transform: "translate(0, -50%)",
				},
			}}
		>
			<Avatar
				src={avatarUrl}
				alt={username || "User avatar"}
				sx={{ width: 36, height: 36, flexShrink: 0 }}
			>
				{(username || "A").charAt(0).toUpperCase()}
			</Avatar>
			<Button
				className="logout-reveal"
				size="small"
				color="error"
				variant="outlined"
				onClick={() => {
					window.location.href = "/api/logout";
				}}
				sx={{
					position: "absolute",
					left: "100%",
					top: "50%",
					ml: 1,
					px: 1.75,
					borderRadius: 999,
					textTransform: "none",
					bgcolor: "background.paper",
					opacity: 0,
					pointerEvents: "none",
					transform: "translate(-8px, -50%)",
					transition: "opacity 0.2s ease, transform 0.2s ease",
					whiteSpace: "nowrap",
					"&:hover": { bgcolor: "error.main", color: "error.contrastText" },
					// Invisible bridge spanning the gap back to the avatar so the
					// pointer never crosses dead space and the reveal stays open.
					"&::before": {
						content: '""',
						position: "absolute",
						top: 0,
						bottom: 0,
						right: "100%",
						width: 16,
					},
				}}
			>
				Logout
			</Button>
		</Box>
	);
};

export default AuthButtons;
