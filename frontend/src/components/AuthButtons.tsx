import React from "react";

const AuthButtons: React.FC = () => {
	const [username, setUsername] = React.useState<string>("");
	const [loggedIn, setLoggedIn] = React.useState<boolean>(false);

	React.useEffect(() => {
		let mounted = true;
		fetch("/api/me")
			.then((response) => response.json())
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
							window.location.href = "/logout";
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
