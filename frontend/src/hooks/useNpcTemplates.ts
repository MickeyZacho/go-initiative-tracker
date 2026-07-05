import { useState, useCallback } from "react";
import { apiGetArray } from "../lib/http";

export interface NpcTemplate {
	ID: number;
	Name: string;
}

export function useNpcTemplates() {
	const [npcTemplates, setNpcTemplates] = useState<NpcTemplate[]>([]);
	const [selectedAddNpcId, setSelectedAddNpcId] = useState<number>(0);

	const fetchNpcTemplates = useCallback(async () => {
		try {
			const data = await apiGetArray<NpcTemplate>("/npcs/templates");
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
