import React from "react";
import {
	FormControl,
	InputLabel,
	Select,
	MenuItem,
	Button,
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material/Select";

export interface AddCharacterControlProps {
	availableLibraryCharacters: { ID: number; Name: string }[];
	selectedAddCharacterId: number;
	setSelectedAddCharacterId: (id: number) => void;
	addExistingCharacterToEncounter: () => void;
	encounterId: number;
}

export const AddCharacterControl: React.FC<AddCharacterControlProps> = ({
	availableLibraryCharacters,
	selectedAddCharacterId,
	setSelectedAddCharacterId,
	addExistingCharacterToEncounter,
	encounterId,
}) => {
	return (
		<>
			<FormControl sx={{ minWidth: 200 }}>
				<InputLabel id="add-existing-character-label">
					Add Existing Character
				</InputLabel>
				<Select
					labelId="add-existing-character-label"
					value={
						selectedAddCharacterId
							? String(selectedAddCharacterId)
							: ""
					}
					label="Add Existing Character"
					onChange={(event: SelectChangeEvent) =>
						setSelectedAddCharacterId(Number(event.target.value))
					}
				>
					{availableLibraryCharacters.map((libChar) => (
						<MenuItem key={libChar.ID} value={String(libChar.ID)}>
							{libChar.Name}
						</MenuItem>
					))}
				</Select>
			</FormControl>
			<Button
				variant="outlined"
				onClick={addExistingCharacterToEncounter}
				disabled={
					!encounterId ||
					!selectedAddCharacterId ||
					availableLibraryCharacters.length === 0
				}
			>
				Add to Encounter
			</Button>
		</>
	);
};
