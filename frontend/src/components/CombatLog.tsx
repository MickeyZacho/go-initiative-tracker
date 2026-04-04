import React from "react";
import {
	Stack,
	Typography,
	FormControl,
	InputLabel,
	Select,
	MenuItem,
	TextField,
	Button,
	Box,
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material/Select";
import type { Character } from "./CharacterList";
import type { LedgerEntry } from "../hooks/useCombatLog";

export interface CombatLogProps {
	ledgerEntries: LedgerEntry[];
	characters: Character[];
	logActorId: number;
	setLogActorId: (id: number) => void;
	logTargetId: number;
	setLogTargetId: (id: number) => void;
	logActionType: string;
	setLogActionType: (type: string) => void;
	logHPChange: string;
	setLogHPChange: (val: string) => void;
	logDescription: string;
	setLogDescription: (val: string) => void;
	addLogEntry: () => void;
	encounterId: number;
	formatLogTime: (timestamp: string) => string;
	characterNameByID: (id: number) => string;
}

export const CombatLog: React.FC<CombatLogProps> = ({
	ledgerEntries,
	characters,
	logActorId,
	setLogActorId,
	logTargetId,
	setLogTargetId,
	logActionType,
	setLogActionType,
	logHPChange,
	setLogHPChange,
	logDescription,
	setLogDescription,
	addLogEntry,
	encounterId,
	formatLogTime,
	characterNameByID,
}) => (
	<Stack spacing={1.5} mt={2}>
		<Typography variant="h6" fontWeight={700}>
			Combat Log
		</Typography>
		<Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
			<FormControl sx={{ minWidth: 140 }}>
				<InputLabel id="log-actor-label">Actor</InputLabel>
				<Select
					labelId="log-actor-label"
					label="Actor"
					value={logActorId ? String(logActorId) : ""}
					onChange={(event: SelectChangeEvent) =>
						setLogActorId(Number(event.target.value))
					}
				>
					{characters.map((character) => (
						<MenuItem
							key={character.ID}
							value={String(character.ID)}
						>
							{character.Name}
						</MenuItem>
					))}
				</Select>
			</FormControl>
			<FormControl sx={{ minWidth: 140 }}>
				<InputLabel id="log-target-label">Target</InputLabel>
				<Select
					labelId="log-target-label"
					label="Target"
					value={String(logTargetId)}
					onChange={(event: SelectChangeEvent) =>
						setLogTargetId(Number(event.target.value))
					}
				>
					<MenuItem value="0">None</MenuItem>
					{characters.map((character) => (
						<MenuItem
							key={character.ID}
							value={String(character.ID)}
						>
							{character.Name}
						</MenuItem>
					))}
				</Select>
			</FormControl>
			<FormControl sx={{ minWidth: 140 }}>
				<InputLabel id="log-action-label">Action</InputLabel>
				<Select
					labelId="log-action-label"
					label="Action"
					value={logActionType}
					onChange={(event: SelectChangeEvent) =>
						setLogActionType(event.target.value)
					}
				>
					<MenuItem value="attack">Attack</MenuItem>
					<MenuItem value="heal">Heal</MenuItem>
					<MenuItem value="note">Note</MenuItem>
				</Select>
			</FormControl>
			<TextField
				label="HP Change"
				type="number"
				value={logHPChange}
				onChange={(event) => setLogHPChange(event.target.value)}
				sx={{ width: 120 }}
			/>
			<TextField
				label="Description"
				value={logDescription}
				onChange={(event) => setLogDescription(event.target.value)}
				sx={{ minWidth: 220, flex: 1 }}
			/>
			<Button
				variant="contained"
				onClick={addLogEntry}
				disabled={!encounterId || !logActorId}
			>
				Add Log
			</Button>
		</Stack>
		<Box
			sx={{
				maxHeight: 180,
				overflowY: "auto",
				border: 1,
				borderColor: "divider",
				borderRadius: 1,
				p: 1,
			}}
		>
			{ledgerEntries.length === 0 ? (
				<Typography color="text.secondary">
					No combat log entries yet.
				</Typography>
			) : (
				ledgerEntries.map((entry) => (
					<Typography key={entry.id} variant="body2" sx={{ mb: 0.5 }}>
						{`[${formatLogTime(entry.created_at)}] `}
						{entry.actor_name || characterNameByID(entry.actor_id)}
						{entry.target_id && entry.target_id > 0
							? ` -> ${entry.target_name || characterNameByID(entry.target_id)}`
							: ""}
						{` [${entry.action_type}] ${entry.hp_change} ${entry.description}`}
					</Typography>
				))
			)}
		</Box>
	</Stack>
);
