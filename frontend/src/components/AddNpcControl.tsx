import React from "react";
import {
	FormControl,
	InputLabel,
	Select,
	MenuItem,
	Button,
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material/Select";

export interface AddNpcControlProps {
	npcTemplates: { ID: number; Name: string }[];
	selectedAddNpcId: number;
	setSelectedAddNpcId: (id: number) => void;
	addNpcToEncounter: () => void;
	encounterId: number;
}

export const AddNpcControl: React.FC<AddNpcControlProps> = ({
	npcTemplates,
	selectedAddNpcId,
	setSelectedAddNpcId,
	addNpcToEncounter,
	encounterId,
}) => {
	return (
		<>
			<FormControl sx={{ minWidth: 200 }}>
				<InputLabel id="add-npc-label">Add NPC</InputLabel>
				<Select
					labelId="add-npc-label"
					value={selectedAddNpcId ? String(selectedAddNpcId) : ""}
					label="Add NPC"
					onChange={(event: SelectChangeEvent) =>
						setSelectedAddNpcId(Number(event.target.value))
					}
				>
					{npcTemplates.map((npc) => (
						<MenuItem key={npc.ID} value={String(npc.ID)}>
							{npc.Name}
						</MenuItem>
					))}
				</Select>
			</FormControl>
			<Button
				variant="outlined"
				onClick={addNpcToEncounter}
				disabled={
					!encounterId ||
					!selectedAddNpcId ||
					npcTemplates.length === 0
				}
			>
				Add NPC
			</Button>
		</>
	);
};
