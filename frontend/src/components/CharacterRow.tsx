import React from "react";
import {
	Alert,
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
import AddIcon from "@mui/icons-material/Add";
import BoltIcon from "@mui/icons-material/Bolt";
import DeleteOutlineIcon from "@mui/icons-material/DeleteOutline";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import type { Character, ConditionInfo } from "./CharacterList";
import { tint } from "../theme";
import { applyHpInput } from "../lib/hp";
import { DeathSaves } from "./DeathSaves";
import { concentrationDC } from "../lib/concentration";

type EditableFieldName =
	| "Name"
	| "ArmorClass"
	| "CurrentHP"
	| "MaxHP"
	| "Initiative"
	| "ToHitModifier"
	| "TempHP";

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
		note?: string,
	) => void;
	onRemoveCondition: (conditionID: number) => void;
	onSetReaction: (characterID: number, used: boolean) => void;
	// Player view: no roster/turn controls, and NPC rows are read-only.
	playerView?: boolean;
	onSetDeathSaves: (
		characterID: number,
		successes: number,
		failures: number,
	) => void;
}

const HEALTH_COLOR: Record<
	string,
	"success" | "warning" | "error" | "default"
> = {
	Healthy: "success",
	Wounded: "warning",
	Bloodied: "error",
	Down: "default",
};

function formatModifier(n: number): string {
	return n >= 0 ? `+${n}` : String(n);
}

// Columns shared by every row and the list header (CharacterList), so from
// `sm` up the stats line up as a table with a single header instead of a
// caption above every value. Below `sm` the row wraps and shows StatLabels.
export const ROW_COLUMNS =
	"minmax(140px, 1fr) 52px 116px 52px 52px 60px 92px 108px";
export const ROW_HEADERS = [
	"Name",
	"AC",
	"HP",
	"Temp",
	"Init",
	"To-Hit",
	"Reaction",
	"",
] as const;

// Per-row caption, shown only on narrow screens where there's no header row.
function StatLabel({ children }: { children: React.ReactNode }) {
	return (
		<Typography
			component="div"
			variant="caption"
			color="text.secondary"
			sx={{
				display: { xs: "block", sm: "none" },
				fontWeight: 600,
				lineHeight: 1.2,
				mb: 0.25,
			}}
		>
			{children}
		</Typography>
	);
}

// Thin bar under the HP numbers: the state of the fight at a glance. Thresholds
// follow the table's "bloodied" convention (half HP).
function HpBar({ current, max }: { current: number; max: number }) {
	const ratio = max > 0 ? Math.max(0, Math.min(1, current / max)) : 0;
	const color =
		ratio > 0.5 ? "success.main" : ratio > 0.25 ? "warning.main" : "error.main";
	return (
		<Box
			aria-hidden
			sx={{
				mt: 0.5,
				mx: "auto",
				width: "100%",
				maxWidth: 104,
				height: 4,
				borderRadius: 2,
				bgcolor: "action.selected",
				overflow: "hidden",
			}}
		>
			<Box
				sx={{
					height: "100%",
					width: `${ratio * 100}%`,
					bgcolor: color,
					transition: "width 0.3s, background-color 0.3s",
				}}
			/>
		</Box>
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
	readOnly?: boolean;
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
	readOnly,
}) =>
	readOnly ? (
		<Box
			component="span"
			sx={{
				display: "inline-block",
				px: 0.75,
				fontWeight: strong ? 600 : 500,
				fontSize: strong ? "1.05rem" : "1rem",
				whiteSpace: "nowrap",
			}}
		>
			{value}
		</Box>
	) : isEditing ? (
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
				// The editable hint only appears while hovering the row, so a table
				// of numbers doesn't read as a wall of underlines.
				borderBottom: "1px dashed transparent",
				".character-row:hover &, &:focus-visible": {
					borderBottomColor: "text.disabled",
				},
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
	playerView = false,
	onSetDeathSaves,
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
	// Spell being concentrated on, stored as the condition's note.
	const [newNote, setNewNote] = React.useState("");
	// Set when a concentrating creature takes damage: prompts the CON save.
	const [concentrationCheck, setConcentrationCheck] = React.useState<{
		dc: number;
		droppedToZero: boolean;
	} | null>(null);

	const conditions = character.Conditions ?? [];
	const concentrating = conditions.find((c) => c.Condition === "Concentrating");
	// Redacted NPCs (player view) carry a HealthStatus instead of HP numbers.
	const redacted = Boolean(character.Redacted);
	const isDown = redacted
		? character.HealthStatus === "Down"
		: character.CurrentHP === 0;
	// A downed NPC (or a PC who failed three death saves) is out of the fight
	// and fades; a dying PC stays in full color while saves are being rolled.
	const isOut =
		isDown &&
		(character.Type === "npc" || (character.DeathSaveFailures ?? 0) >= 3);
	// Players can't edit NPCs; the server enforces this too.
	const locked = playerView && character.Type === "npc";
	// Only offer conditions not already applied — except leveled ones (Exhaustion),
	// which stay in the picker so the DM can raise or lower the level. Adding one
	// again upserts the existing row rather than stacking a second chip.
	const availableConditions = conditionCatalog.filter(
		(info) =>
			info.MaxLevel > 0 || !conditions.some((c) => c.Condition === info.Name),
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
		setNewNote("");
	};

	const submitCondition = () => {
		if (!newCondition) return;
		const trimmed = newDuration.trim();
		const rounds = trimmed === "" ? null : Math.floor(Number(trimmed));
		if (rounds !== null && (!Number.isFinite(rounds) || rounds <= 0)) return;
		// The backend rejects a level on a binary condition, so only send one when
		// the selected condition actually has levels.
		const level = maxLevel > 0 ? Number(newLevel) : null;
		if (
			level !== null &&
			(!Number.isFinite(level) || level < 1 || level > maxLevel)
		)
			return;
		onAddCondition(
			character.ID,
			newCondition,
			rounds,
			level,
			newCondition === "Concentrating" ? newNote.trim() : "",
		);
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
			const temp = character.TempHP ?? 0;
			const next = applyHpInput(
				editing.value,
				{ current: character.CurrentHP, temp },
				character.MaxHP,
			);
			setEditing({ field: null, value: "" });
			if (
				next === null ||
				(next.current === character.CurrentHP && next.temp === temp)
			)
				return;
			// Damage to a concentrating creature calls for a Constitution save
			// (or ends concentration outright at 0 HP). Temp HP still counts: the
			// creature took the hit.
			const damage = character.CurrentHP + temp - (next.current + next.temp);
			if (damage > 0 && concentrating) {
				setConcentrationCheck({
					dc: concentrationDC(damage),
					droppedToZero: next.current === 0,
				});
			}
			const updated = {
				...character,
				CurrentHP: next.current,
				TempHP: next.temp,
			};
			setCharacters((prev) =>
				prev.map((c) => (c.ID === character.ID ? updated : c)),
			);
			onSave(updated);
			return;
		} else if (field === "TempHP") {
			// Temp HP doesn't go negative; blank/garbage means none.
			newValue = Math.max(0, Math.floor(Number(editing.value)) || 0);
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
		readOnly: locked,
	});

	const name = character.Name || "Character";

	// Bring the creature whose turn it is into view when the turn reaches it, so
	// the DM never hunts for the current row in a long list.
	const rowRef = React.useRef<HTMLDivElement>(null);
	React.useEffect(() => {
		if (!character.IsActive || !rowRef.current) return;
		const reduceMotion = window.matchMedia?.(
			"(prefers-reduced-motion: reduce)",
		).matches;
		rowRef.current.scrollIntoView?.({
			block: "nearest",
			behavior: reduceMotion ? "auto" : "smooth",
		});
	}, [character.IsActive]);

	return (
		<Paper
			ref={rowRef}
			variant="outlined"
			className={`character-row${character.IsActive ? " active" : ""}`}
			aria-current={character.IsActive ? "true" : undefined}
			sx={{
				px: { xs: 1.5, sm: 2 },
				py: 1.25,
				// Leave room above the list so scrollIntoView doesn't tuck the row
				// under the sticky app bar.
				scrollMarginTop: 96,
				scrollMarginBottom: 96,
				// Whose turn it is: a warm wash plus a solid left edge. Two signals
				// are enough; the combat bar also names the active creature.
				borderLeftWidth: 4,
				borderLeftColor: isOut
					? "text.disabled"
					: character.IsActive
						? "primary.main"
						: "divider",
				bgcolor: character.IsActive
					? tint("primary", 0.12)
					: "background.paper",
				filter: isOut ? "grayscale(1)" : "none",
				opacity: isOut ? 0.6 : 1,
				transition: "background-color 0.2s, border-color 0.2s, opacity 0.2s",
			}}
		>
			<Box
				sx={{
					display: { xs: "flex", sm: "grid" },
					gridTemplateColumns: ROW_COLUMNS,
					flexWrap: "wrap",
					alignItems: { xs: "flex-end", sm: "center" },
					columnGap: { xs: 1.5, sm: 1 },
					rowGap: 1,
				}}
			>
				{/* Name */}
				<Box sx={{ flex: "1 1 150px", minWidth: 0 }}>
					<StatLabel>Name</StatLabel>
					<Box sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
						{isDown && (
							<Tooltip
								title={
									character.Type === "npc"
										? "Down — skipped in the turn order"
										: "Down (0 HP)"
								}
							>
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
						{...fieldProps(
							"ArmorClass",
							character.ArmorClass,
							redacted ? "—" : character.ArmorClass,
						)}
						type="number"
						width={56}
						ariaLabel={`${name} AC`}
					/>
				</Box>

				{/* HP */}
				<Box sx={{ textAlign: "center" }}>
					<StatLabel>HP</StatLabel>
					{redacted ? (
						<Chip
							size="small"
							label={character.HealthStatus || "?"}
							color={HEALTH_COLOR[character.HealthStatus ?? ""] ?? "default"}
							variant="outlined"
							sx={{ fontWeight: 600 }}
						/>
					) : (
						<Box>
							<Box
								sx={{
									display: "flex",
									alignItems: "center",
									justifyContent: "center",
									gap: 0.5,
								}}
							>
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
							<HpBar current={character.CurrentHP} max={character.MaxHP} />
						</Box>
					)}
				</Box>

				{/* Temporary HP: soaks damage before HP (the -N input drains it
				    first). Hidden for redacted NPCs, but the cell stays so the grid
				    columns don't shift. */}
				<Box sx={{ textAlign: "center" }}>
					{!redacted && (
						<>
							<StatLabel>Temp</StatLabel>
							<Box
								sx={{
									color:
										(character.TempHP ?? 0) > 0 ? "info.main" : "text.disabled",
								}}
							>
								<EditableField
									{...fieldProps("TempHP", character.TempHP ?? 0)}
									type="number"
									width={56}
									ariaLabel={`${name} temporary HP`}
									hint="Temporary HP — absorbs damage first, doesn't stack"
								/>
							</Box>
						</>
					)}
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
							redacted ? "—" : formatModifier(character.ToHitModifier ?? 0),
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
						icon={<BoltIcon fontSize="small" />}
						label={character.ReactionUsed ? "Used" : "Ready"}
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
						onClick={
							locked
								? undefined
								: () => onSetReaction(character.ID, !character.ReactionUsed)
						}
						sx={
							character.ReactionUsed
								? {
										color: "text.disabled",
										textDecoration: "line-through",
										"& .MuiChip-icon": { color: "text.disabled" },
									}
								: {
										bgcolor: tint("primary", 0.14),
										color: "primary.main",
										"& .MuiChip-icon": { color: "primary.main" },
										"&:hover": { bgcolor: tint("primary", 0.24) },
									}
						}
					/>
				</Box>

				{/* Row actions: add condition, give turn, remove. */}
				<Box
					sx={{
						ml: "auto",
						justifySelf: "end",
						display: "flex",
						alignItems: "center",
					}}
				>
					{!locked && !addingCondition && availableConditions.length > 0 && (
						<Tooltip title="Add condition">
							<IconButton
								size="small"
								aria-label={`Add condition to ${name}`}
								onClick={() => setAddingCondition(true)}
								sx={{ color: "text.secondary" }}
							>
								<AddIcon fontSize="small" />
							</IconButton>
						</Tooltip>
					)}
					{/* Give turn (hidden on the creature whose turn it already is) */}
					{!playerView && !character.IsActive && (
						<Tooltip title="Give this creature the turn">
							<IconButton
								size="small"
								aria-label={`Give ${name} the turn`}
								onClick={giveTurn}
								sx={{ color: "text.secondary" }}
							>
								<PlayArrowIcon fontSize="small" />
							</IconButton>
						</Tooltip>
					)}
					{!playerView && (
						<Tooltip title="Remove from encounter">
							<IconButton
								size="small"
								aria-label={`Remove ${name}`}
								onClick={onRemove}
								sx={{
									color: "text.disabled",
									"&:hover": {
										color: "error.main",
										bgcolor: tint("error", 0.1),
									},
								}}
							>
								<DeleteOutlineIcon fontSize="small" />
							</IconButton>
						</Tooltip>
					)}
				</Box>
			</Box>

			{concentrationCheck && concentrating && (
				<Alert
					severity={concentrationCheck.droppedToZero ? "error" : "warning"}
					sx={{ mt: 1, py: 0 }}
					action={
						<>
							<Button
								color="inherit"
								size="small"
								onClick={() => {
									onRemoveCondition(concentrating.ID);
									setConcentrationCheck(null);
								}}
							>
								{concentrationCheck.droppedToZero ? "End it" : "Lost it"}
							</Button>
							{!concentrationCheck.droppedToZero && (
								<Button
									color="inherit"
									size="small"
									onClick={() => setConcentrationCheck(null)}
								>
									Kept it
								</Button>
							)}
						</>
					}
				>
					{concentrationCheck.droppedToZero
						? `${name} dropped to 0 HP — concentration${concentrating.Note ? ` on ${concentrating.Note}` : ""} ends.`
						: `${name} is concentrating${concentrating.Note ? ` on ${concentrating.Note}` : ""}: Constitution save DC ${concentrationCheck.dc}.`}
				</Alert>
			)}

			{/* A PC at 0 HP is dying, not dead: track their death saves. */}
			{isDown && character.Type !== "npc" && !redacted && (
				<DeathSaves
					name={name}
					successes={character.DeathSaveSuccesses ?? 0}
					failures={character.DeathSaveFailures ?? 0}
					onChange={(s, f) => onSetDeathSaves(character.ID, s, f)}
				/>
			)}

			{/* Conditions strip: only takes space when there's something on it
			    (the "+" in the row actions opens the picker). */}
			{(conditions.length > 0 || addingCondition) && (
				<Box
					sx={{
						display: "flex",
						alignItems: "center",
						flexWrap: "wrap",
						gap: 0.75,
						mt: 1,
					}}
				>
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
							{newCondition === "Concentrating" && (
								<TextField
									size="small"
									value={newNote}
									onChange={(e) => setNewNote(e.target.value)}
									placeholder="Spell (optional)"
									slotProps={{ htmlInput: { "aria-label": "Spell" } }}
									sx={{ width: 150 }}
								/>
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
					) : null}
				</Box>
			)}
		</Paper>
	);
};
