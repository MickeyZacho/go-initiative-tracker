import React from "react";
import {
	Autocomplete,
	Box,
	Button,
	FormControlLabel,
	Switch,
	TextField,
	Typography,
} from "@mui/material";
import { useNpcTemplateSearch, type NpcTemplate } from "../hooks/useNpcTemplates";

export const MAX_NPC_COPIES = 20;

// The "generate names" choice is a per-browser preference. Storage can be
// unavailable (private mode, blocked site data), so fall back to off.
const NAMES_KEY = "addNpc.generateNames";
const loadGenerateNames = () => {
	try {
		return localStorage.getItem(NAMES_KEY) === "1";
	} catch {
		return false;
	}
};
const saveGenerateNames = (on: boolean) => {
	try {
		localStorage.setItem(NAMES_KEY, on ? "1" : "0");
	} catch {
		// Not remembered; harmless.
	}
};

export interface AddNpcControlProps {
	selectedNpc: NpcTemplate | null;
	setSelectedNpc: (npc: NpcTemplate | null) => void;
	addNpcToEncounter: (count: number, generateNames: boolean) => void;
	encounterId: number;
}

// Searchable template picker plus a quantity, so "four goblins" is one action.
// Options come from the server as you type (your own templates when empty),
// so the full SRD list is never loaded. The backend numbers the copies
// (Goblin 1–4), optionally gives each a nickname ("Goblin 3 · Snagtooth"),
// and rolls each one's initiative.
export const AddNpcControl: React.FC<AddNpcControlProps> = ({
	selectedNpc,
	setSelectedNpc,
	addNpcToEncounter,
	encounterId,
}) => {
	const [count, setCount] = React.useState("1");
	const [input, setInput] = React.useState("");
	const [generateNames, setGenerateNames] = React.useState(loadGenerateNames);
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
			<FormControlLabel
				control={
					<Switch
						checked={generateNames}
						onChange={(e) => {
							setGenerateNames(e.target.checked);
							saveGenerateNames(e.target.checked);
						}}
					/>
				}
				label="Names"
				title='Give each copy a nickname, e.g. "Goblin 3 · Snagtooth"'
			/>
			<Button
				variant="outlined"
				onClick={() => {
					addNpcToEncounter(parsed, generateNames);
					setCount("1");
				}}
				disabled={!encounterId || !selectedNpc || !validCount}
			>
				Add NPC{validCount && parsed > 1 ? ` ×${parsed}` : ""}
			</Button>
		</>
	);
};
