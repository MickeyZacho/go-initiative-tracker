import { Box, Button, Paper, Stack, Typography } from "@mui/material";

export interface CombatBarProps {
	round: number;
	activeName: string | null;
	hasCombatants: boolean;
	busy: boolean;
	onStart: () => void;
	onNext: () => void;
	onReset: () => void;
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
				spacing={2}
				useFlexGap
				flexWrap="wrap"
			>
				<Box sx={{ minWidth: 0, flex: "1 1 160px" }}>
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
						<Button
							color="inherit"
							size="small"
							onClick={onReset}
							disabled={busy}
						>
							End combat
						</Button>
						<Button
							variant="contained"
							size="large"
							onClick={onNext}
							disabled={busy || !hasCombatants}
							title="Next turn (Space)"
						>
							Next turn ▶
						</Button>
					</>
				) : (
					<>
						{/* Before the fight: roll every NPC at once (d20 + DEX), so
						    only the players' rolls need typing in. */}
						{hasNpcs && (
							<Button
								size="small"
								onClick={onRollNpcInitiative}
								disabled={busy}
							>
								🎲 Roll NPC initiative
							</Button>
						)}
						<Button
							variant="contained"
							size="large"
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
