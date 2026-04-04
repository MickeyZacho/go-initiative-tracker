import React from "react";
import {
	Stack,
	FormControl,
	InputLabel,
	Select,
	MenuItem,
	TextField,
	Button,
	Typography,
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material/Select";
import type { Character } from "./CharacterList";

export interface CombatControlsProps {
	characters: Character[];
	quickActionByActor: Record<number, { targetId: number; amount: string }>;
	handleQuickActionChange: (
		actorID: number,
		field: "targetId" | "amount",
		value: number | string,
	) => void;
	handleQuickAmountKeyDown: (
		event: React.KeyboardEvent<HTMLElement>,
		actor: Character,
	) => void;
	applyQuickAction: (actor: Character, actionType: "attack" | "heal") => void;
	combatStarted: boolean;
}

export const CombatControls: React.FC<CombatControlsProps> = ({
	characters,
	quickActionByActor,
	handleQuickActionChange,
	handleQuickAmountKeyDown,
	applyQuickAction,
	combatStarted,
}) => {
	if (!combatStarted) return null;
	return (
		<Stack spacing={2}>
			{[...characters]
				.sort((a, b) => b.Initiative - a.Initiative)
				.map((character) => {
					const quickConfig = quickActionByActor[character.ID];
					const quickTargetID = quickConfig?.targetId ?? 0;
					const quickAmountRaw = quickConfig?.amount ?? "1";
					const quickAmount = Math.floor(Number(quickAmountRaw));
					const targetCharacter = characters.find(
						(c) => c.ID === quickTargetID,
					);
					const hasValidTarget = Boolean(targetCharacter);
					const hasValidAmount =
						Number.isFinite(quickAmount) && quickAmount > 0;
					const attackPreviewHP = targetCharacter
						? Math.max(0, targetCharacter.CurrentHP - quickAmount)
						: 0;
					const healPreviewHP = targetCharacter
						? Math.min(
								targetCharacter.MaxHP,
								targetCharacter.CurrentHP + quickAmount,
							)
						: 0;
					const rowValidationMessage = !hasValidTarget
						? "Pick a valid target"
						: !hasValidAmount
							? "Amount must be greater than 0"
							: "";

					return (
						<div style={{ width: "100%" }} key={character.ID}>
							<Stack
								direction="row"
								spacing={1}
								useFlexGap
								flexWrap="wrap"
								alignItems="center"
								mt={1}
							>
								<FormControl
									size="small"
									sx={{ minWidth: 140 }}
								>
									<InputLabel
										id={`target-label-${character.ID}`}
									>
										Target
									</InputLabel>
									<Select
										size="small"
										labelId={`target-label-${character.ID}`}
										label="Target"
										value={String(
											quickActionByActor[character.ID]
												?.targetId ?? 0,
										)}
										onChange={(event: SelectChangeEvent) =>
											handleQuickActionChange(
												character.ID,
												"targetId",
												Number(event.target.value),
											)
										}
									>
										{characters.map((targetChar) => (
											<MenuItem
												key={targetChar.ID}
												value={String(targetChar.ID)}
											>
												{targetChar.Name}
											</MenuItem>
										))}
									</Select>
								</FormControl>
								<TextField
									size="small"
									type="number"
									label="Amount"
									value={quickAmountRaw}
									onChange={(event) =>
										handleQuickActionChange(
											character.ID,
											"amount",
											event.target.value,
										)
									}
									onKeyDown={(event) =>
										handleQuickAmountKeyDown(
											event,
											character,
										)
									}
									sx={{ width: 100 }}
								/>
								<Button
									size="small"
									color="error"
									variant="contained"
									onClick={() =>
										applyQuickAction(character, "attack")
									}
									disabled={
										!hasValidTarget || !hasValidAmount
									}
								>
									Attack
								</Button>
								<Button
									size="small"
									color="success"
									variant="contained"
									onClick={() =>
										applyQuickAction(character, "heal")
									}
									disabled={
										!hasValidTarget || !hasValidAmount
									}
								>
									Heal
								</Button>
							</Stack>
							<Typography
								variant="caption"
								color={
									rowValidationMessage
										? "error"
										: "text.secondary"
								}
								sx={{ display: "block", mt: 0.5 }}
							>
								{rowValidationMessage
									? rowValidationMessage
									: targetCharacter
										? `Preview: ${targetCharacter.Name} HP ${targetCharacter.CurrentHP} → ${attackPreviewHP} (Attack) / ${healPreviewHP} (Heal)  • Enter = Attack, Shift+Enter = Heal`
										: "Select a target to preview result"}
							</Typography>
						</div>
					);
				})}
		</Stack>
	);
};
