import { useState } from "react";
import {
	Box,
	Button,
	IconButton,
	Paper,
	Stack,
	TextField,
	Tooltip,
	Typography,
} from "@mui/material";
import VisibilityIcon from "@mui/icons-material/Visibility";
import VisibilityOffIcon from "@mui/icons-material/VisibilityOff";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutline";
import type { Character } from "./CharacterList";

export interface HiddenCombatantsProps {
	waiting: Character[];
	round: number;
	onReveal: (character: Character) => void;
	// null = no schedule; the DM reveals by hand.
	onSchedule: (character: Character, revealRound: number | null) => void;
	onRemove: (character: Character) => void;
}

// The DM's staging area: creatures already in the encounter but invisible to
// players and out of the turn order until revealed, by hand or automatically
// when their arrival round begins. Players never receive these rows.
export function HiddenCombatants({
	waiting,
	round,
	onReveal,
	onSchedule,
	onRemove,
}: HiddenCombatantsProps) {
	if (waiting.length === 0) return null;
	return (
		<Paper
			variant="outlined"
			sx={{ p: 1.5, borderStyle: "dashed", bgcolor: "transparent" }}
		>
			<Stack direction="row" alignItems="center" spacing={1} mb={1}>
				<VisibilityOffIcon fontSize="small" color="disabled" />
				<Typography variant="subtitle2" fontWeight={600}>
					Waiting in the wings ({waiting.length})
				</Typography>
				<Typography variant="caption" color="text.secondary">
					Hidden from players, no turns until revealed
				</Typography>
			</Stack>
			<Stack spacing={0.5}>
				{[...waiting]
					.sort(
						(a, b) =>
							(a.RevealRound ?? Infinity) - (b.RevealRound ?? Infinity) ||
							a.ID - b.ID,
					)
					.map((c) => (
						<WaitingRow
							key={c.ID}
							character={c}
							round={round}
							onReveal={onReveal}
							onSchedule={onSchedule}
							onRemove={onRemove}
						/>
					))}
			</Stack>
		</Paper>
	);
}

function WaitingRow({
	character,
	round,
	onReveal,
	onSchedule,
	onRemove,
}: {
	character: Character;
	round: number;
	onReveal: (c: Character) => void;
	onSchedule: (c: Character, r: number | null) => void;
	onRemove: (c: Character) => void;
}) {
	const saved = character.RevealRound ? String(character.RevealRound) : "";
	const [draft, setDraft] = useState(saved);
	// Re-sync when another viewer (or the auto-reveal) changes the schedule.
	const [lastSaved, setLastSaved] = useState(saved);
	if (saved !== lastSaved) {
		setLastSaved(saved);
		setDraft(saved);
	}

	const commit = () => {
		const trimmed = draft.trim();
		if (trimmed === saved) return;
		if (trimmed === "") {
			onSchedule(character, null);
			return;
		}
		const n = Math.floor(Number(trimmed));
		if (!Number.isFinite(n) || n < 1) {
			setDraft(saved);
			return;
		}
		onSchedule(character, n);
	};

	const initiative = character.Initiative
		? `Init ${character.Initiative}`
		: "Rolls on reveal";

	return (
		<Box
			sx={{
				display: "flex",
				alignItems: "center",
				flexWrap: "wrap",
				gap: 1,
				py: 0.5,
				px: 1,
				borderRadius: 1,
				"&:hover": { bgcolor: "action.hover" },
			}}
		>
			<Box sx={{ flex: "1 1 140px", minWidth: 0 }}>
				<Typography noWrap fontWeight={500}>
					{character.Name}
				</Typography>
				<Typography variant="caption" color="text.secondary">
					{character.Type === "npc" ? "NPC" : "Character"} · {initiative}
				</Typography>
			</Box>
			<TextField
				label="Arrives round"
				type="number"
				size="small"
				value={draft}
				placeholder="Manual"
				onChange={(e) => setDraft(e.target.value)}
				onBlur={commit}
				onKeyDown={(e) => {
					if (e.key === "Enter") (e.target as HTMLInputElement).blur();
				}}
				helperText={
					character.RevealRound && round > 0 && character.RevealRound <= round
						? "Next round"
						: undefined
				}
				slotProps={{
					htmlInput: { min: 1 },
					inputLabel: { shrink: true },
				}}
				sx={{ width: 130 }}
			/>
			<Button
				size="small"
				variant="outlined"
				startIcon={<VisibilityIcon />}
				onClick={() => onReveal(character)}
			>
				Reveal now
			</Button>
			<Tooltip title="Remove from encounter">
				<IconButton
					size="small"
					aria-label={`Remove ${character.Name}`}
					onClick={() => onRemove(character)}
					sx={{ color: "text.disabled", "&:hover": { color: "error.main" } }}
				>
					<DeleteOutlineIcon fontSize="small" />
				</IconButton>
			</Tooltip>
		</Box>
	);
}
