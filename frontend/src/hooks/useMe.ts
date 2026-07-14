import { useEffect, useState } from "react";
import { apiGet } from "../lib/http";

export interface Me {
	loggedIn: boolean;
	username: string;
	discordID: string;
	avatar: string;
}

const LOGGED_OUT: Me = {
	loggedIn: false,
	username: "",
	discordID: "",
	avatar: "",
};

// useMe fetches the signed-in user from /me once on mount. Used to gate
// owner-only UI (e.g. encounter sharing) on the caller's Discord id.
export function useMe(): Me {
	const [me, setMe] = useState<Me>(LOGGED_OUT);

	useEffect(() => {
		let mounted = true;
		apiGet<Partial<Me>>("/me")
			.then((data) => {
				if (!mounted) return;
				setMe({
					loggedIn: Boolean(data.loggedIn),
					username: data.username || "",
					discordID: data.discordID || "",
					avatar: data.avatar || "",
				});
			})
			.catch(() => {
				if (mounted) setMe(LOGGED_OUT);
			});
		return () => {
			mounted = false;
		};
	}, []);

	return me;
}
