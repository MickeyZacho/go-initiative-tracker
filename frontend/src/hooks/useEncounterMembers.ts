import { useCallback, useState } from "react";
import { apiGet, apiPost } from "../lib/http";
import type { Friend } from "./useFriends";

interface MembersResponse {
	members?: Friend[];
}

// useEncounterMembers loads and mutates the shared-edit member list for a single
// encounter. It is stateful per open encounter; call load(encounterId) when the
// owner opens the sharing panel, then add/remove refresh that same encounter.
export function useEncounterMembers() {
	const [members, setMembers] = useState<Friend[]>([]);
	const [error, setError] = useState("");

	const load = useCallback(async (encounterId: number) => {
		setError("");
		try {
			const res = await apiGet<MembersResponse>(
				`/encounters/members?encounter_id=${encounterId}`,
			);
			setMembers(res.members ?? []);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to load members",
			);
			setMembers([]);
		}
	}, []);

	const addMember = useCallback(
		async (encounterId: number, userId: string) => {
			await apiPost(
				"/encounters/members/add",
				{ encounter_id: encounterId, user_id: userId },
				"Failed to add member",
			);
			await load(encounterId);
		},
		[load],
	);

	const removeMember = useCallback(
		async (encounterId: number, userId: string) => {
			await apiPost(
				"/encounters/members/remove",
				{ encounter_id: encounterId, user_id: userId },
				"Failed to remove member",
			);
			await load(encounterId);
		},
		[load],
	);

	const clear = useCallback(() => {
		setMembers([]);
		setError("");
	}, []);

	return { members, error, load, addMember, removeMember, clear };
}
