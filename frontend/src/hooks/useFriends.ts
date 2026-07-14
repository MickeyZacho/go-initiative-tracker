import { useCallback, useEffect, useState } from "react";
import { apiGet, apiPost } from "../lib/http";

export interface Friend {
	discord_id: string;
	username: string;
	avatar: string;
}

interface FriendsResponse {
	friends?: Friend[];
}

interface RequestsResponse {
	incoming?: Friend[];
	outgoing?: Friend[];
}

// useFriends manages the caller's friends list and pending requests, exposing
// the mutations (send/accept/decline/remove) that each refresh the relevant
// lists. All calls go through the shared http helpers, so the Discord cookie is
// sent and the { status: "success" } envelope is enforced on mutations.
export function useFriends() {
	const [friends, setFriends] = useState<Friend[]>([]);
	const [incoming, setIncoming] = useState<Friend[]>([]);
	const [outgoing, setOutgoing] = useState<Friend[]>([]);
	const [error, setError] = useState("");
	const [loading, setLoading] = useState(false);

	const refresh = useCallback(async () => {
		setLoading(true);
		setError("");
		try {
			const [friendsRes, requestsRes] = await Promise.all([
				apiGet<FriendsResponse>("/friends"),
				apiGet<RequestsResponse>("/friends/requests"),
			]);
			setFriends(friendsRes.friends ?? []);
			setIncoming(requestsRes.incoming ?? []);
			setOutgoing(requestsRes.outgoing ?? []);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to load friends",
			);
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		refresh();
	}, [refresh]);

	const sendRequest = useCallback(
		async (username: string) => {
			await apiPost(
				"/friends/request",
				{ username: username.trim() },
				"Failed to send friend request",
			);
			await refresh();
		},
		[refresh],
	);

	const accept = useCallback(
		async (discordID: string) => {
			await apiPost(
				"/friends/accept",
				{ discord_id: discordID },
				"Failed to accept request",
			);
			await refresh();
		},
		[refresh],
	);

	const decline = useCallback(
		async (discordID: string) => {
			await apiPost(
				"/friends/decline",
				{ discord_id: discordID },
				"Failed to decline request",
			);
			await refresh();
		},
		[refresh],
	);

	const remove = useCallback(
		async (discordID: string) => {
			await apiPost(
				"/friends/remove",
				{ discord_id: discordID },
				"Failed to remove friend",
			);
			await refresh();
		},
		[refresh],
	);

	return {
		friends,
		incoming,
		outgoing,
		error,
		loading,
		refresh,
		sendRequest,
		accept,
		decline,
		remove,
	};
}
