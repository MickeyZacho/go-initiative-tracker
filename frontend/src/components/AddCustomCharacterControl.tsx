import React, { useState } from "react";
import {
	Box,
	Button,
	FormControl,
	InputLabel,
	MenuItem,
	Select,
	Stack,
	TextField,
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material/Select";

export interface CustomCharacterDraft {
	name: string;
	type: string;
	maxHP: number;
	armorClass: number;
	initiative: number;
	toHitModifier: number;
}

export interface AddCustomCharacterControlProps {
	addCustomCharacterToEncounter: (draft: CustomCharacterDraft) => Promise<void>;
	encounterId: number;
}

const emptyDraft = {
	name: "",
	type: "npc",
	maxHP: "10",
	armorClass: "10",
	initiative: "0",
	toHitModifier: "0",
};

// A lightweight inline form for spinning up a one-off character/NPC and dropping
// it straight into the current encounter, without first saving it to the library
// view. The backend's /save-character (id=0 + encounter_id) both creates the
// character and adds it to the encounter in one call.
export const AddCustomCharacterControl: React.FC<
	AddCustomCharacterControlProps
> = ({ addCustomCharacterToEncounter, encounterId }) => {
	const [draft, setDraft] = useState(emptyDraft);
	const [submitting, setSubmitting] = useState(false);

	const handleText =
		(field: keyof typeof emptyDraft) =>
		(event: React.ChangeEvent<HTMLInputElement>) =>
			setDraft((prev) => ({ ...prev, [field]: event.target.value }));

	const handleType = (event: SelectChangeEvent) =>
		setDraft((prev) => ({ ...prev, type: event.target.value }));

	const canSubmit =
		!!encounterId &&
		draft.name.trim().length > 0 &&
		Number(draft.maxHP) >= 1 &&
		!submitting;

	const submit = async () => {
		if (!canSubmit) return;
		setSubmitting(true);
		try {
			await addCustomCharacterToEncounter({
				name: draft.name.trim(),
				type: draft.type,
				maxHP: Number(draft.maxHP),
				armorClass: Number(draft.armorClass) || 0,
				initiative: Number(draft.initiative) || 0,
				toHitModifier: Number(draft.toHitModifier) || 0,
			});
			setDraft(emptyDraft);
		} finally {
			setSubmitting(false);
		}
	};

	return (
		<Stack
			direction="row"
			spacing={2}
			alignItems="center"
			flexWrap="wrap"
			useFlexGap
		>
			<TextField
				label="Name"
				value={draft.name}
				onChange={handleText("name")}
				size="small"
				sx={{ minWidth: 160 }}
				onKeyDown={(e) => {
					if (e.key === "Enter") void submit();
				}}
			/>
			<FormControl size="small" sx={{ minWidth: 110 }}>
				<InputLabel id="custom-character-type-label">Type</InputLabel>
				<Select
					labelId="custom-character-type-label"
					value={draft.type}
					label="Type"
					onChange={handleType}
				>
					<MenuItem value="npc">NPC</MenuItem>
					<MenuItem value="pc">Character</MenuItem>
				</Select>
			</FormControl>
			<TextField
				label="Max HP"
				type="number"
				value={draft.maxHP}
				onChange={handleText("maxHP")}
				size="small"
				sx={{ width: 100 }}
			/>
			<TextField
				label="AC"
				type="number"
				value={draft.armorClass}
				onChange={handleText("armorClass")}
				size="small"
				sx={{ width: 90 }}
			/>
			<TextField
				label="Initiative"
				type="number"
				value={draft.initiative}
				onChange={handleText("initiative")}
				size="small"
				sx={{ width: 100 }}
			/>
			<TextField
				label="To Hit"
				type="number"
				value={draft.toHitModifier}
				onChange={handleText("toHitModifier")}
				size="small"
				sx={{ width: 90 }}
			/>
			<Box>
				<Button
					variant="outlined"
					onClick={() => void submit()}
					disabled={!canSubmit}
				>
					Add Custom
				</Button>
			</Box>
		</Stack>
	);
};
