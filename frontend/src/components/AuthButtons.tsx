import React from "react";
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

	const avatarUrl = discordID && avatar
		? `https://cdn.discordapp.com/avatars/${discordID}/${avatar}.png`
		: null;

	return (
		<div style={{ marginBottom: "1rem" }}>
			{loggedIn ? (
				<>
					{avatarUrl && (
						<img
							src={avatarUrl}
							alt="avatar"
							style={{ width: 32, height: 32, borderRadius: "50%", marginRight: "0.5rem", verticalAlign: "middle" }}
						/>
					)}
					<span style={{ marginRight: "1rem" }}>
						Welcome, {username || "Adventurer"}!
					</span>
					<button
						onClick={() => {
							window.location.href = "/api/logout";
						}}
					>
						Logout
					</button>
				</>
			) : (
				<button
					onClick={() =>
						(window.location.href = "/api/login/discord")
					}
				>
					Login with Discord
				</button>
			)}
		</div>
	);
};

export default AuthButtons;
