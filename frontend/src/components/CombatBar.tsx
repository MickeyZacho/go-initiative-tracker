import { Box, Button, Paper, Stack, Typography } from "@mui/material";
import CasinoIcon from "@mui/icons-material/Casino";
import SkipNextIcon from "@mui/icons-material/SkipNext";

export interface CombatBarProps {
	round: number;
	activeName: string | null;
	hasCombatants: boolean;
	busy: boolean;
	onStart: () => void;
	onNext: () => void;
	onReset: () => void;
	// False in the player view: players can end their turn, nothing else.
	canControl: boolean;
	hasNpcs: boolean;
	onRollNpcInitiative: () => void;
}

// The DM's primary control surface: round, whose turn it is, and the buttons to
// run the fight. Sticks to the bottom of the viewport so "Next" stays in thumb
// reach on a tablet however long the initiative list gets. Space also advances.
export function CombatBar({
	round,
	activeName,
	hasCombatants,
	busy,
	onStart,
	onNext,
	onReset,
	canControl,
	hasNpcs,
	onRollNpcInitiative,
}: CombatBarProps) {
	const started = round > 0;
	return (
		<Paper
			elevation={6}
			sx={{
				position: "sticky",
				bottom: 12,
				zIndex: (t) => t.zIndex.appBar - 1,
				px: { xs: 1.5, sm: 2 },
				py: 1,
				border: 1,
				borderColor: "divider",
			}}
		>
			<Stack
				direction="row"
				alignItems="center"
				spacing={{ xs: 1, sm: 2 }}
			>
				<Box sx={{ minWidth: 0, flex: "1 1 0" }}>
					<Typography
						variant="overline"
						color="text.secondary"
						sx={{ lineHeight: 1.2, display: "block" }}
					>
						{started ? `Round ${round}` : "Combat not started"}
					</Typography>
					<Typography fontWeight={700} noWrap>
						{activeName ? `${activeName}'s turn` : "—"}
					</Typography>
				</Box>
				{started ? (
					<>
						{canControl && (
							<Button
								color="inherit"
								size="small"
								onClick={onReset}
								disabled={busy}
							>
								End combat
							</Button>
						)}
						<Button
							variant="contained"
							size="large"
							sx={{ whiteSpace: "nowrap", flexShrink: 0 }}
							onClick={onNext}
							disabled={busy || !hasCombatants}
							title="Next turn (Space)"
							endIcon={<SkipNextIcon />}
						>
							Next turn
						</Button>
					</>
				) : !canControl ? (
					<Typography variant="body2" color="text.secondary">
						Waiting for the DM to start combat
					</Typography>
				) : (
					<>
						{/* Before the fight: roll every NPC at once (d20 + DEX), so
						    only the players' rolls need typing in. */}
						{hasNpcs && (
							<Button
								size="small"
								onClick={onRollNpcInitiative}
								startIcon={<CasinoIcon />}
								aria-label="Roll NPC initiative"
								sx={{
									whiteSpace: "nowrap",
									// Icon-only on phones so the main button keeps its room.
									minWidth: 0,
									"& .MuiButton-startIcon": { mr: { xs: 0, sm: 1 } },
								}}
								disabled={busy}
							>
								<Box component="span" sx={{ display: { xs: "none", sm: "inline" } }}>
									Roll NPC initiative
								</Box>
							</Button>
						)}
						<Button
							variant="contained"
							size="large"
							sx={{ whiteSpace: "nowrap", flexShrink: 0 }}
							onClick={onStart}
							disabled={busy || !hasCombatants}
						>
							Start combat
						</Button>
					</>
				)}
			</Stack>
		</Paper>
	);
}
