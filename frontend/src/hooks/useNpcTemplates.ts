import { useState, useCallback } from "react";
import { parseJsonResponse } from "../lib/http";

export interface NpcTemplate {
	ID: number;
	Name: string;
}

export function useNpcTemplates() {
	const [npcTemplates, setNpcTemplates] = useState<NpcTemplate[]>([]);
	const [selectedAddNpcId, setSelectedAddNpcId] = useState<number>(0);

	const fetchNpcTemplates = useCallback(async () => {
		try {
			const response = await fetch("/api/npcs/templates", {
				credentials: "include",
			});
			if (!response.ok) throw new Error("Failed to fetch NPC templates");
			const payload = await parseJsonResponse<unknown>(response);
			const data: NpcTemplate[] = Array.isArray(payload) ? payload : [];
			setNpcTemplates(data);
			if (data.length > 0) {
				setSelectedAddNpcId(data[0].ID);
			}
		} catch {
			setNpcTemplates([]);
		}
	}, []);

	return {
		npcTemplates,
		selectedAddNpcId,
		setSelectedAddNpcId,
		fetchNpcTemplates,
	};
}
