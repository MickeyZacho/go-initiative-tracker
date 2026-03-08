import React from "react";

function getDiscordUsername(): string | null {
	const cookie = document.cookie
		.split("; ")
		.find((row) => row.startsWith("discord_user="));
	return cookie ? decodeURIComponent(cookie.split("=")[1]) : null;
}

const AuthButtons: React.FC = () => {
	const username = getDiscordUsername();
	const loggedIn = !!username;

	return (
		<div style={{ marginBottom: "1rem" }}>
			{loggedIn ? (
				<>
					<span style={{ marginRight: "1rem" }}>
						Welcome, {username}!
					</span>
					<button
						onClick={() => {
							window.location.href = "/logout";
							setTimeout(() => window.location.reload(), 500);
						}}
					>
						Logout
					</button>
				</>
			) : (
				<button
					onClick={() => (window.location.href = "/login/discord")}
				>
					Login with Discord
				</button>
			)}
		</div>
	);
};

export default AuthButtons;
