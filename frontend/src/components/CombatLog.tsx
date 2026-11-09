import React from "react";
import { Typography, Box } from "@mui/material";
import type { LedgerEntry } from "../hooks/useCombatLog";

export interface CombatLogProps {
	ledgerEntries: LedgerEntry[];
	formatLogTime: (timestamp: string) => string;
	characterNameByID: (id: number) => string;
}

// Turn a single ledger entry into its human-readable sentence. The backend writes
// entries automatically as combat happens (turns, damage, healing, conditions,
// reactions) and stores the structured fields; we compose the wording here so the
// log reads naturally. Unknown/legacy action types fall back to their stored
// description, which also covers any seeded demo rows.
function describeEntry(
	entry: LedgerEntry,
	nameFor: (id: number) => string,
): string {
	const actor = entry.actor_name || nameFor(entry.actor_id);
	const magnitude = Math.abs(entry.hp_change);
	switch (entry.action_type) {
		case "turn":
			return `${actor}'s turn`;
		case "damage":
			return `${actor} took ${magnitude} damage`;
		case "heal":
			return `${actor} recovered ${magnitude} HP`;
		case "condition":
			return `${actor} gained ${entry.description || "a condition"}`;
		case "reaction":
			return `${actor} used a reaction`;
		default:
			return entry.description || `${actor}: ${entry.action_type}`;
	}
}

export const CombatLog: React.FC<CombatLogProps> = ({
	ledgerEntries,
	formatLogTime,
	characterNameByID,
}) => (
	<Box
		sx={{
			maxHeight: 220,
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
					<Box
						component="span"
						sx={{ color: "text.secondary", mr: 1 }}
					>
						{formatLogTime(entry.created_at)}
					</Box>
					{describeEntry(entry, characterNameByID)}
				</Typography>
			))
		)}
	</Box>
);
