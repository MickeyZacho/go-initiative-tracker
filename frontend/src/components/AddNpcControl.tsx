import React from "react";
import {
	FormControl,
	InputLabel,
	Select,
	MenuItem,
	Button,
	TextField,
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material/Select";

export const MAX_NPC_COPIES = 20;

export interface AddNpcControlProps {
	npcTemplates: { ID: number; Name: string }[];
	selectedAddNpcId: number;
	setSelectedAddNpcId: (id: number) => void;
	addNpcToEncounter: (count: number) => void;
	encounterId: number;
}

// Template picker plus a quantity, so "four goblins" is one action. The backend
// numbers the copies (Goblin 1–4) and rolls each one's initiative.
export const AddNpcControl: React.FC<AddNpcControlProps> = ({
	npcTemplates,
	selectedAddNpcId,
	setSelectedAddNpcId,
	addNpcToEncounter,
	encounterId,
}) => {
	const [count, setCount] = React.useState("1");
	const parsed = Math.floor(Number(count));
	const validCount =
		Number.isFinite(parsed) && parsed >= 1 && parsed <= MAX_NPC_COPIES;

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
			<TextField
				label="How many"
				type="number"
				value={count}
				onChange={(e) => setCount(e.target.value)}
				error={!validCount}
				slotProps={{ htmlInput: { min: 1, max: MAX_NPC_COPIES } }}
				sx={{ width: 110 }}
			/>
			<Button
				variant="outlined"
				onClick={() => {
					addNpcToEncounter(parsed);
					setCount("1");
				}}
				disabled={
					!encounterId ||
					!selectedAddNpcId ||
					npcTemplates.length === 0 ||
					!validCount
				}
			>
				Add NPC{validCount && parsed > 1 ? ` ×${parsed}` : ""}
			</Button>
		</>
	);
};
