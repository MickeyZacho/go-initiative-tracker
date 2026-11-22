import React from "react";
import { Autocomplete, Box, Button, TextField, Typography } from "@mui/material";
import { useNpcTemplateSearch, type NpcTemplate } from "../hooks/useNpcTemplates";

export const MAX_NPC_COPIES = 20;

export interface AddNpcControlProps {
	selectedNpc: NpcTemplate | null;
	setSelectedNpc: (npc: NpcTemplate | null) => void;
	addNpcToEncounter: (count: number) => void;
	encounterId: number;
}

// Searchable template picker plus a quantity, so "four goblins" is one action.
// Options come from the server as you type (your own templates when empty),
// so the full SRD list is never loaded. The backend numbers the copies
// (Goblin 1–4) and rolls each one's initiative.
export const AddNpcControl: React.FC<AddNpcControlProps> = ({
	selectedNpc,
	setSelectedNpc,
	addNpcToEncounter,
	encounterId,
}) => {
	const [count, setCount] = React.useState("1");
	const [input, setInput] = React.useState("");
	const { results, loading } = useNpcTemplateSearch(input);
	const parsed = Math.floor(Number(count));
	const validCount =
		Number.isFinite(parsed) && parsed >= 1 && parsed <= MAX_NPC_COPIES;

	return (
		<>
			<Autocomplete
				sx={{ minWidth: 240 }}
				options={results}
				value={selectedNpc}
				onChange={(_, npc) => setSelectedNpc(npc)}
				inputValue={input}
				onInputChange={(_, value) => setInput(value)}
				// The server already filtered; don't filter again client-side.
				filterOptions={(options) => options}
				getOptionLabel={(npc) => npc.Name}
				isOptionEqualToValue={(a, b) => a.ID === b.ID}
				loading={loading}
				noOptionsText={
					input.trim() ? "No matching NPCs" : "No NPC templates yet"
				}
				renderOption={(props, npc) => {
					const { key, ...rest } = props;
					return (
						<li key={key} {...rest}>
							<Box>
								<Typography variant="body2">{npc.Name}</Typography>
								{npc.Description && (
									<Typography variant="caption" color="text.secondary">
										{npc.Description}
									</Typography>
								)}
							</Box>
						</li>
					);
				}}
				renderInput={(params) => (
					<TextField {...params} label="Add NPC" placeholder="Search monsters…" />
				)}
			/>
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
				disabled={!encounterId || !selectedNpc || !validCount}
			>
				Add NPC{validCount && parsed > 1 ? ` ×${parsed}` : ""}
			</Button>
		</>
	);
};
