import React from "react";
import { parseJsonResponse } from "../lib/http";

const AuthButtons: React.FC = () => {
	const [username, setUsername] = React.useState<string>("");
	const [loggedIn, setLoggedIn] = React.useState<boolean>(false);

	React.useEffect(() => {
		let mounted = true;
		fetch("/api/me", { credentials: "include" })
			.then((response) =>
				parseJsonResponse<{ loggedIn?: boolean; username?: string }>(
					response,
				),
			)
			.then((data) => {
				if (!mounted) return;
				setLoggedIn(Boolean(data.loggedIn));
				setUsername(data.username || "");
			})
			.catch(() => {
				if (!mounted) return;
				setLoggedIn(false);
				setUsername("");
			});
		return () => {
			mounted = false;
		};
	}, []);

	return (
		<div style={{ marginBottom: "1rem" }}>
			{loggedIn ? (
				<>
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
