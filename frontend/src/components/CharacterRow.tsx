import React from "react";
import {
	Box,
	Button,
	Chip,
	IconButton,
	InputBase,
	Paper,
	TextField,
	Tooltip,
	Typography,
} from "@mui/material";
import type { Character, ConditionInfo } from "./CharacterList";
import { tint } from "../theme";
import { applyHpInput } from "../lib/hp";

type EditableFieldName =
	| "Name"
	| "ArmorClass"
	| "CurrentHP"
	| "MaxHP"
	| "Initiative"
	| "ToHitModifier";

interface CharacterRowProps {
	character: Character;
	setCharacters: React.Dispatch<React.SetStateAction<Character[]>>;
	// Explicitly hand the turn to this creature (moves the server's turn pointer).
	onGiveTurn: (id: number) => void;
	onSave: (character: Character) => void;
	onRemove: () => void;
	conditionCatalog: ConditionInfo[];
	onAddCondition: (
		characterID: number,
		condition: string,
		durationRounds: number | null,
		level: number | null,
	) => void;
	onRemoveCondition: (conditionID: number) => void;
	onSetReaction: (characterID: number, used: boolean) => void;
}

function formatModifier(n: number): string {
	return n >= 0 ? `+${n}` : String(n);
}

// Small caption above each stat, so the row reads like a stat line on any width.
function StatLabel({ children }: { children: React.ReactNode }) {
	return (
		<Typography
			component="div"
			variant="caption"
			color="primary"
			sx={{ fontWeight: 600, lineHeight: 1.2, mb: 0.25 }}
		>
			{children}
		</Typography>
	);
}

// Click-to-edit value: shows the value with a dashed underline, swaps to an
// input while editing. Enter commits, Escape cancels (handled by onKeyDown).
const EditableField: React.FC<{
	value: string | number;
	type: "text" | "number";
	isEditing: boolean;
	onStartEdit: () => void;
	onChange: (v: string) => void;
	onBlur: () => void;
	onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
	width: number;
	ariaLabel: string;
	strong?: boolean;
	placeholder?: string;
	hint?: string;
}> = ({
	value,
	type,
	isEditing,
	onStartEdit,
	onChange,
	onBlur,
	onKeyDown,
	width,
	ariaLabel,
	strong,
	placeholder,
	hint,
}) =>
	isEditing ? (
		<InputBase
			type={type}
			value={String(value)}
			placeholder={placeholder}
			title={hint}
			autoFocus
			onChange={(e) => onChange(e.target.value)}
			onBlur={onBlur}
			onKeyDown={onKeyDown}
			inputProps={{ "aria-label": ariaLabel }}
			sx={{
				width,
				px: 0.75,
				border: 1,
				borderColor: "primary.main",
				borderRadius: 1,
				fontWeight: strong ? 600 : 500,
				fontSize: strong ? "1.05rem" : "1rem",
			}}
		/>
	) : (
		<Box
			component="span"
			role="button"
			tabIndex={0}
			aria-label={`${ariaLabel}: ${value}. Click to edit`}
			onClick={onStartEdit}
			onFocus={onStartEdit}
			title={hint}
			sx={{
				display: "inline-block",
				minWidth: 28,
				maxWidth: width + 40,
				px: 0.75,
				borderRadius: 1,
				borderBottom: "1px dashed",
				borderColor: "primary.main",
				cursor: "text",
				fontWeight: strong ? 600 : 500,
				fontSize: strong ? "1.05rem" : "1rem",
				whiteSpace: "nowrap",
				overflow: "hidden",
				textOverflow: "ellipsis",
				verticalAlign: "bottom",
				"&:hover": { bgcolor: "action.hover" },
			}}
		>
			{value}
		</Box>
	);

export const CharacterRow: React.FC<CharacterRowProps> = ({
	character,
	setCharacters,
	onGiveTurn,
	onSave,
	onRemove,
	conditionCatalog,
	onAddCondition,
	onRemoveCondition,
	onSetReaction,
}) => {
	const [editing, setEditing] = React.useState<{
		field: EditableFieldName | null;
		value: string;
	}>({ field: null, value: "" });
	// Add-condition form state; the picker is hidden behind a "+" toggle so the
	// row stays uncluttered until the user wants to apply a condition.
	const [addingCondition, setAddingCondition] = React.useState(false);
	const [newCondition, setNewCondition] = React.useState("");
	const [newDuration, setNewDuration] = React.useState("");
	const [newLevel, setNewLevel] = React.useState("1");

	const conditions = character.Conditions ?? [];
	const isDown = character.CurrentHP === 0;
	// Only offer conditions not already applied — except leveled ones (Exhaustion),
	// which stay in the picker so the DM can raise or lower the level. Adding one
	// again upserts the existing row rather than stacking a second chip.
	const availableConditions = conditionCatalog.filter(
		(info) =>
			info.MaxLevel > 0 ||
			!conditions.some((c) => c.Condition === info.Name),
	);
	const selectedInfo = conditionCatalog.find(
		(info) => info.Name === newCondition,
	);
	const maxLevel = selectedInfo?.MaxLevel ?? 0;

	// Picking a leveled condition pre-fills the next level up from whatever the
	// creature already has, since exhaustion almost always goes up by one.
	const handleConditionChange = (name: string) => {
		setNewCondition(name);
		const info = conditionCatalog.find((c) => c.Name === name);
		if (info && info.MaxLevel > 0) {
			const current = conditions.find((c) => c.Condition === name)?.Level ?? 0;
			setNewLevel(String(Math.min(current + 1, info.MaxLevel)));
		}
	};

	const resetConditionForm = () => {
		setAddingCondition(false);
		setNewCondition("");
		setNewDuration("");
		setNewLevel("1");
	};

	const submitCondition = () => {
		if (!newCondition) return;
		const trimmed = newDuration.trim();
		const rounds = trimmed === "" ? null : Math.floor(Number(trimmed));
		if (rounds !== null && (!Number.isFinite(rounds) || rounds <= 0)) return;
		// The backend rejects a level on a binary condition, so only send one when
		// the selected condition actually has levels.
		const level = maxLevel > 0 ? Number(newLevel) : null;
		if (level !== null && (!Number.isFinite(level) || level < 1 || level > maxLevel))
			return;
		onAddCondition(character.ID, newCondition, rounds, level);
		resetConditionForm();
	};

	const startEdit = (field: EditableFieldName, value: string | number) => {
		// HP opens empty (current value as placeholder) so the DM can type "-7"
		// straight away instead of clearing the old number first.
		setEditing({ field, value: field === "CurrentHP" ? "" : String(value) });
	};

	const handleFieldBlur = (field: EditableFieldName) => {
		if (editing.value.trim() === "") {
			setEditing({ field: null, value: "" });
			return;
		}
		let newValue: string | number = editing.value;
		if (field === "CurrentHP") {
			const next = applyHpInput(
				editing.value,
				character.CurrentHP,
				character.MaxHP,
			);
			setEditing({ field: null, value: "" });
			if (next === null || next === character.CurrentHP) return;
			newValue = next;
		} else if (field !== "Name") {
			newValue = Number(editing.value);
			if (isNaN(newValue)) newValue = 0;
		}
		const updated = { ...character, [field]: newValue };
		setCharacters((prev) =>
			prev.map((c) => (c.ID === character.ID ? updated : c)),
		);
		setEditing({ field: null, value: "" });
		onSave(updated);
	};

	const handleFieldKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
		if (e.key === "Enter") {
			(e.target as HTMLInputElement).blur();
		} else if (e.key === "Escape") {
			setEditing({ field: null, value: "" });
		}
	};

	// Giving the turn is a deliberate action, not a side effect of clicking the
	// row: a stray click while editing HP must never move initiative for everyone.
	// Flip IsActive locally for instant feedback (keeping identity for rows that
	// don't change), then persist.
	const giveTurn = () => {
		setCharacters((prev) =>
			prev.map((c) => {
				const shouldBeActive = c.ID === character.ID;
				return c.IsActive === shouldBeActive
					? c
					: { ...c, IsActive: shouldBeActive };
			}),
		);
		onGiveTurn(character.ID);
	};

	// Shared props for the click-to-edit fields. `display` is how the value reads
	// when not editing (e.g. "+3" for a modifier); editing uses the raw value.
	const fieldProps = (
		field: EditableFieldName,
		current: string | number,
		display: string | number = current,
	) => ({
		value: editing.field === field ? editing.value : display,
		isEditing: editing.field === field,
		onStartEdit: () => startEdit(field, current),
		onChange: (v: string) => setEditing((ed) => ({ ...ed, value: v })),
		onBlur: () => handleFieldBlur(field),
		onKeyDown: handleFieldKeyDown,
	});

	const name = character.Name || "Character";

	return (
		<Paper
			variant="outlined"
			className={`character-row${character.IsActive ? " active" : ""}`}
			sx={{
				px: { xs: 1.5, sm: 2 },
				py: 1.25,
				borderLeftWidth: 5,
				borderColor: character.IsActive ? "primary.main" : "divider",
				borderLeftColor: isDown
					? "text.disabled"
					: character.IsActive
						? "primary.main"
						: tint("primary", 0.35),
				bgcolor: character.IsActive
					? tint("primary", 0.08)
					: "background.paper",
				filter: isDown ? "grayscale(1)" : "none",
				opacity: isDown ? 0.75 : 1,
				transition: "background-color 0.2s, border-color 0.2s, opacity 0.2s",
			}}
		>
			<Box
				sx={{
					display: "flex",
					flexWrap: "wrap",
					alignItems: "flex-end",
					columnGap: { xs: 1.5, sm: 2.5 },
					rowGap: 1,
				}}
			>
				{/* Name */}
				<Box sx={{ flex: "1 1 150px", minWidth: 0 }}>
					<StatLabel>Name</StatLabel>
					<Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
						{isDown && (
							<Tooltip title="Down (0 HP)">
								<Box
									component="span"
									aria-label="Down"
									sx={{
										display: "inline-flex",
										alignItems: "center",
										justifyContent: "center",
										width: 22,
										height: 22,
										borderRadius: 1,
										bgcolor: "text.secondary",
										color: "background.paper",
										fontWeight: 700,
										fontSize: "0.85rem",
										flexShrink: 0,
									}}
								>
									✕
								</Box>
							</Tooltip>
						)}
						<EditableField
							{...fieldProps("Name", character.Name)}
							type="text"
							width={150}
							ariaLabel="Name"
							strong
						/>
					</Box>
				</Box>

				{/* Armor Class */}
				<Box sx={{ textAlign: "center" }}>
					<StatLabel>AC</StatLabel>
					<EditableField
						{...fieldProps("ArmorClass", character.ArmorClass)}
						type="number"
						width={56}
						ariaLabel={`${name} AC`}
					/>
				</Box>

				{/* HP */}
				<Box sx={{ textAlign: "center" }}>
					<StatLabel>HP</StatLabel>
					<Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
						<EditableField
							{...fieldProps("CurrentHP", character.CurrentHP)}
							type="text"
							width={64}
							ariaLabel={`${name} current HP`}
							placeholder={String(character.CurrentHP)}
							hint="Type -7 for damage, +5 to heal, or a new total"
						/>
						<Typography component="span" color="text.secondary">
							/
						</Typography>
						<EditableField
							{...fieldProps("MaxHP", character.MaxHP)}
							type="number"
							width={56}
							ariaLabel={`${name} max HP`}
						/>
					</Box>
				</Box>

				{/* Initiative */}
				<Box sx={{ textAlign: "center" }}>
					<StatLabel>Init</StatLabel>
					<EditableField
						{...fieldProps("Initiative", character.Initiative)}
						type="number"
						width={56}
						ariaLabel={`${name} initiative`}
					/>
				</Box>

				{/* To-hit modifier, so the DM can resolve an attack without leaving
				    the combat screen. */}
				<Box sx={{ textAlign: "center" }}>
					<StatLabel>To-Hit</StatLabel>
					<EditableField
						{...fieldProps(
							"ToHitModifier",
							character.ToHitModifier ?? 0,
							formatModifier(character.ToHitModifier ?? 0),
						)}
						type="number"
						width={56}
						ariaLabel={`${name} to-hit modifier`}
					/>
				</Box>

				{/* Reaction toggle. A creature has one reaction per round; click to
				    mark it spent, click again to restore. The backend also clears it
				    automatically at the start of the creature's turn. */}
				<Box sx={{ textAlign: "center" }}>
					<StatLabel>Reaction</StatLabel>
					<Chip
						size="small"
						label={character.ReactionUsed ? "⚡ Used" : "⚡ Ready"}
						color={character.ReactionUsed ? "default" : "primary"}
						variant={character.ReactionUsed ? "outlined" : "filled"}
						aria-pressed={character.ReactionUsed}
						aria-label={
							character.ReactionUsed
								? `${name} reaction used — click to restore`
								: `${name} reaction available — click to mark used`
						}
						title={
							character.ReactionUsed
								? "Reaction used — click to restore"
								: "Reaction available — click to mark used"
						}
						onClick={() =>
							onSetReaction(character.ID, !character.ReactionUsed)
						}
						sx={{
							fontWeight: 600,
							textDecoration: character.ReactionUsed
								? "line-through"
								: "none",
						}}
					/>
				</Box>

				{/* Give turn (hidden on the creature whose turn it already is) */}
				<Box sx={{ ml: "auto", display: "flex", alignItems: "center" }}>
					{!character.IsActive && (
						<Tooltip title="Give this creature the turn">
							<IconButton
								size="small"
								aria-label={`Give ${name} the turn`}
								onClick={giveTurn}
								sx={{ color: "text.secondary", fontSize: "0.9rem" }}
							>
								▶
							</IconButton>
						</Tooltip>
					)}
					{/* Delete */}
					<Tooltip title="Remove from encounter">
						<IconButton
							size="small"
							aria-label={`Remove ${name}`}
							onClick={onRemove}
							sx={{
								color: "text.disabled",
								"&:hover": { color: "error.main", bgcolor: tint("error", 0.1) },
							}}
						>
							<svg
								aria-hidden="true"
								focusable="false"
								width="16"
								height="16"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								strokeWidth="2"
								strokeLinecap="round"
								strokeLinejoin="round"
							>
								<polyline points="3 6 5 6 21 6" />
								<path d="M19 6l-1 14H6L5 6" />
								<path d="M10 11v6" />
								<path d="M14 11v6" />
								<path d="M9 6V4h6v2" />
							</svg>
						</IconButton>
					</Tooltip>
				</Box>
			</Box>

			{/* Conditions strip */}
			<Box
				sx={{
					display: "flex",
					alignItems: "center",
					flexWrap: "wrap",
					gap: 0.75,
					mt: 1,
					pt: 1,
					borderTop: "1px dashed",
					borderColor: "divider",
				}}
			>
				<Typography variant="caption" color="primary" sx={{ fontWeight: 600 }}>
					Conditions:
				</Typography>
				{conditions.length === 0 && !addingCondition && (
					<Typography variant="caption" color="text.disabled">
						none
					</Typography>
				)}
				{conditions.map((cond) => {
					// For a leveled condition the tooltip explains what that level
					// does (e.g. exhaustion 5 = speed 0); notes come first when the DM
					// wrote one.
					const levelEffect =
						cond.Level != null
							? conditionCatalog.find((i) => i.Name === cond.Condition)
									?.LevelEffects?.[cond.Level - 1]
							: undefined;
					const tooltip =
						[cond.Note, levelEffect].filter(Boolean).join(" — ") || "";
					const label =
						cond.Condition +
						(cond.Level != null ? ` ${cond.Level}` : "") +
						(cond.DurationRounds != null ? ` (${cond.DurationRounds})` : "");
					return (
						<Tooltip key={cond.ID} title={tooltip}>
							<Chip
								size="small"
								label={label}
								onDelete={() => onRemoveCondition(cond.ID)}
								slotProps={{
									root: { "aria-label": label },
								}}
								sx={{
									fontWeight: 600,
									bgcolor: tint("secondary", 0.14),
									color: "secondary.main",
									"& .MuiChip-deleteIcon": { color: "secondary.main" },
								}}
							/>
						</Tooltip>
					);
				})}
				{addingCondition ? (
					<Box
						sx={{
							display: "inline-flex",
							alignItems: "center",
							flexWrap: "wrap",
							gap: 0.75,
						}}
					>
						<TextField
							select
							size="small"
							value={newCondition}
							onChange={(e) => handleConditionChange(e.target.value)}
							autoFocus
							slotProps={{
								select: { native: true },
								htmlInput: { "aria-label": "Condition" },
							}}
							sx={{ minWidth: 130 }}
						>
							<option value="">Select…</option>
							{availableConditions.map((info) => (
								<option key={info.Name} value={info.Name}>
									{info.Name}
								</option>
							))}
						</TextField>
						{maxLevel > 0 && (
							<TextField
								select
								size="small"
								value={newLevel}
								onChange={(e) => setNewLevel(e.target.value)}
								slotProps={{
									select: { native: true },
									htmlInput: { "aria-label": "Level" },
								}}
								sx={{ maxWidth: 220 }}
							>
								{Array.from({ length: maxLevel }, (_, i) => i + 1).map(
									(lvl) => (
										<option key={lvl} value={String(lvl)}>
											{`Lv ${lvl}${
												selectedInfo?.LevelEffects?.[lvl - 1]
													? ` — ${selectedInfo.LevelEffects[lvl - 1]}`
													: ""
											}`}
										</option>
									),
								)}
							</TextField>
						)}
						<TextField
							type="number"
							size="small"
							value={newDuration}
							onChange={(e) => setNewDuration(e.target.value)}
							placeholder="∞"
							title="Duration in rounds (blank = until removed)"
							slotProps={{
								htmlInput: { min: 1, "aria-label": "Duration in rounds" },
							}}
							sx={{ width: 72 }}
						/>
						<Button
							size="small"
							variant="contained"
							onClick={submitCondition}
							disabled={!newCondition}
						>
							Add
						</Button>
						<Button size="small" onClick={resetConditionForm}>
							Cancel
						</Button>
					</Box>
				) : (
					availableConditions.length > 0 && (
						<Chip
							size="small"
							variant="outlined"
							color="primary"
							label="+ Condition"
							aria-label="Add condition"
							onClick={() => setAddingCondition(true)}
							sx={{ borderStyle: "dashed" }}
						/>
					)
				)}
			</Box>
		</Paper>
	);
};
