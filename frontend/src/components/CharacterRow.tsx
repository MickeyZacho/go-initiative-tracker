import React from "react";
import type { Character, ConditionInfo } from "./CharacterList";

type EditableFieldName =
	| "Name"
	| "ArmorClass"
	| "CurrentHP"
	| "MaxHP"
	| "Initiative";

interface CharacterRowProps {
	character: Character;
	setCharacters: React.Dispatch<React.SetStateAction<Character[]>>;
	setSelected: (id: number) => void;
	onSelect: (id: number) => void;
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
}

const EditableField: React.FC<{
	value: string | number;
	type: "text" | "number";
	isEditing: boolean;
	onClick: () => void;
	onFocus: () => void;
	onChange: (v: string) => void;
	onBlur: () => void;
	onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void;
	style?: React.CSSProperties;
	autoFocus?: boolean;
}> = React.memo(
	({
		value,
		type,
		isEditing,
		onClick,
		onFocus,
		onChange,
		onBlur,
		onKeyDown,
		style,
		autoFocus,
	}) => (
		<>
			{isEditing ? (
				<input
					type={type}
					value={String(value)}
					autoFocus={autoFocus}
					onChange={(e) => onChange(e.target.value)}
					onBlur={onBlur}
					onKeyDown={onKeyDown}
					style={style}
				/>
			) : (
				<span
					style={{
						...style,
						border: "1px solid transparent",
						background: "none",
						cursor: "pointer",
						transition: "border 0.2s, background 0.2s",
						borderBottom: "1px dashed #1976d2",
					}}
					tabIndex={0}
					onClick={onClick}
					onFocus={onFocus}
					onMouseEnter={(e) =>
						((e.currentTarget as HTMLSpanElement).style.background =
							"#e3f2fd")
					}
					onMouseLeave={(e) =>
						((e.currentTarget as HTMLSpanElement).style.background =
							"none")
					}
				>
					{value}
				</span>
			)}
		</>
	)
);

export const CharacterRow: React.FC<CharacterRowProps> = ({
	character,
	setCharacters,
	setSelected,
	onSelect,
	onSave,
	onRemove,
	conditionCatalog,
	onAddCondition,
	onRemoveCondition,
}) => {
	const [editing, setEditing] = React.useState<{
		field: EditableFieldName | null;
		value: string;
	}>({ field: null, value: "" });
	const [deleteHovered, setDeleteHovered] = React.useState(false);
	// Add-condition form state; the picker is hidden behind a "+" toggle so the
	// row stays uncluttered until the user wants to apply a condition.
	const [addingCondition, setAddingCondition] = React.useState(false);
	const [newCondition, setNewCondition] = React.useState("");
	const [newDuration, setNewDuration] = React.useState("");
	const [newLevel, setNewLevel] = React.useState("1");

	const conditions = character.Conditions ?? [];
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

	const handleFieldClick = (
		field: EditableFieldName,
		value: string | number
	) => {
		setEditing({ field, value: String(value) });
	};

	const handleFieldBlur = (field: EditableFieldName) => {
		if (editing.value.trim() === "") {
			setEditing({ field: null, value: "" });
			return;
		}
		let newValue: string | number = editing.value;
		if (field !== "Name") {
			newValue = Number(editing.value);
			if (isNaN(newValue)) newValue = 0;
		}
		const updated = { ...character, [field]: newValue };
		setCharacters((prev) =>
			prev.map((c) => (c.ID === character.ID ? updated : c))
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

	const handleRowClick = (e: React.MouseEvent<HTMLDivElement>) => {
		const tag = (e.target as HTMLElement).tagName;
		if (tag === "INPUT") return;
		// Only replace the objects whose IsActive actually flips; keep identity
		// for every other row so React/the compiler can skip re-rendering (and
		// repainting) them. Rebuilding the whole list repaints the entire card,
		// which flickers unrelated elements like the collapsed combat log.
		setCharacters((prev) =>
			prev.map((c) => {
				const shouldBeActive = c.ID === character.ID;
				return c.IsActive === shouldBeActive
					? c
					: { ...c, IsActive: shouldBeActive };
			})
		);
		setSelected(character.ID);
		onSelect(character.ID);
	};

	return (
		<div
			className={`character-row${character.IsActive ? " active" : ""}`}
			style={{
				display: "flex",
				flexDirection: "column",
				alignItems: "stretch",
				borderRadius: 8,
				borderTop: "1px solid " + (character.IsActive ? "#1976d2" : "#e0e0e0"),
				borderRight: "1px solid " + (character.IsActive ? "#1976d2" : "#e0e0e0"),
				borderBottom: "1px solid " + (character.IsActive ? "#1976d2" : "#e0e0e0"),
				borderLeft: character.CurrentHP === 0
					? "5px solid #9e9e9e"
					: character.IsActive
						? "5px solid #1565c0"
						: "5px solid #90caf9",
				background: character.CurrentHP === 0
					? "#f0f0f0"
					: character.IsActive ? "#f0f7ff" : "#fff",
				boxShadow: character.IsActive
					? "0 2px 8px rgba(25,118,210,0.08)"
					: "none",
				filter: character.CurrentHP === 0 ? "grayscale(1)" : "none",
				transition: "background 0.2s, border-color 0.2s, filter 0.2s",
				cursor: "pointer",
				padding: "0.75rem 1rem",
				minHeight: 56,
				width: "100%",
				boxSizing: "border-box",
			}}
			onClick={handleRowClick}
		>
			<div
				style={{
					display: "flex",
					alignItems: "center",
					width: "100%",
				}}
			>
			{/* Name */}
			<div style={{ flex: 2 }}>
				<label
					style={{
						fontWeight: 500,
						color: "#1976d2",
						marginBottom: 4,
						display: "block",
					}}
				>
					Name
				</label>
				<div style={{ display: "flex", alignItems: "center", gap: 4 }}>
				{character.CurrentHP === 0 && (
					<span
						style={{
							display: "inline-flex",
							alignItems: "center",
							justifyContent: "center",
							width: 22,
							height: 22,
							borderRadius: 4,
							background: "#616161",
							color: "#e0e0e0",
							fontWeight: 700,
							fontSize: "0.85rem",
							flexShrink: 0,
						}}
					>
						✕
					</span>
				)}
				<EditableField
					value={
						editing.field === "Name"
							? editing.value
							: character.Name
					}
					type="text"
					isEditing={editing.field === "Name"}
					onClick={() => handleFieldClick("Name", character.Name)}
					onFocus={() => handleFieldClick("Name", character.Name)}
					onChange={(v) => setEditing((ed) => ({ ...ed, value: v }))}
					onBlur={() => handleFieldBlur("Name")}
					onKeyDown={handleFieldKeyDown}
					style={{
						width: 120,
						padding: "4px 8px",
						fontWeight: 600,
						fontSize: "1.1rem",
						color: character.IsActive ? "#1976d2" : "#333",
						borderRadius: 4,
					}}
					autoFocus={editing.field === "Name"}
				/>
				</div>
			</div>
			{/* Armor Class */}
			<div style={{ flex: 1, textAlign: "center" }}>
				<label
					style={{
						fontWeight: 500,
						color: "#1976d2",
						marginBottom: 4,
						display: "block",
					}}
				>
					AC
				</label>
				<EditableField
					value={
						editing.field === "ArmorClass"
							? editing.value
							: character.ArmorClass
					}
					type="number"
					isEditing={editing.field === "ArmorClass"}
					onClick={() =>
						handleFieldClick("ArmorClass", character.ArmorClass)
					}
					onFocus={() =>
						handleFieldClick("ArmorClass", character.ArmorClass)
					}
					onChange={(v) => setEditing((ed) => ({ ...ed, value: v }))}
					onBlur={() => handleFieldBlur("ArmorClass")}
					onKeyDown={handleFieldKeyDown}
					style={{
						width: 60,
						padding: "4px 8px",
						fontWeight: 500,
						borderRadius: 4,
					}}
					autoFocus={editing.field === "ArmorClass"}
				/>
			</div>
			{/* HP */}
			<div
				style={{
					flex: 1,
					textAlign: "center",
					display: "flex",
					alignItems: "center",
					justifyContent: "center",
					flexDirection: "column",
				}}
			>
				<label
					style={{
						fontWeight: 500,
						color: "#1976d2",
						marginBottom: 4,
						display: "block",
					}}
				>
					HP
				</label>
				<div
					style={{
						display: "flex",
						alignItems: "center",
						justifyContent: "center",
					}}
				>
					<EditableField
						value={
							editing.field === "CurrentHP"
								? editing.value
								: character.CurrentHP
						}
						type="number"
						isEditing={editing.field === "CurrentHP"}
						onClick={() =>
							handleFieldClick("CurrentHP", character.CurrentHP)
						}
						onFocus={() =>
							handleFieldClick("CurrentHP", character.CurrentHP)
						}
						onChange={(v) =>
							setEditing((ed) => ({ ...ed, value: v }))
						}
						onBlur={() => handleFieldBlur("CurrentHP")}
						onKeyDown={handleFieldKeyDown}
						style={{
							width: 60,
							padding: "4px 8px",
							fontWeight: 500,
							borderRadius: 4,
							marginRight: 8,
						}}
						autoFocus={editing.field === "CurrentHP"}
					/>
					<span style={{ margin: "0 8px" }}>/</span>
					<EditableField
						value={
							editing.field === "MaxHP"
								? editing.value
								: character.MaxHP
						}
						type="number"
						isEditing={editing.field === "MaxHP"}
						onClick={() =>
							handleFieldClick("MaxHP", character.MaxHP)
						}
						onFocus={() =>
							handleFieldClick("MaxHP", character.MaxHP)
						}
						onChange={(v) =>
							setEditing((ed) => ({ ...ed, value: v }))
						}
						onBlur={() => handleFieldBlur("MaxHP")}
						onKeyDown={handleFieldKeyDown}
						style={{
							width: 60,
							padding: "4px 8px",
							fontWeight: 500,
							borderRadius: 4,
							marginLeft: 8,
						}}
						autoFocus={editing.field === "MaxHP"}
					/>
				</div>
			</div>
			{/* Initiative */}
			<div style={{ flex: 1, textAlign: "center" }}>
				<label
					style={{
						fontWeight: 500,
						color: "#1976d2",
						marginBottom: 4,
						display: "block",
					}}
				>
					Initiative
				</label>
				<EditableField
					value={
						editing.field === "Initiative"
							? editing.value
							: character.Initiative
					}
					type="number"
					isEditing={editing.field === "Initiative"}
					onClick={() =>
						handleFieldClick("Initiative", character.Initiative)
					}
					onFocus={() =>
						handleFieldClick("Initiative", character.Initiative)
					}
					onChange={(v) => setEditing((ed) => ({ ...ed, value: v }))}
					onBlur={() => handleFieldBlur("Initiative")}
					onKeyDown={handleFieldKeyDown}
					style={{
						width: 60,
						padding: "4px 8px",
						fontWeight: 500,
						borderRadius: 4,
					}}
					autoFocus={editing.field === "Initiative"}
				/>
			</div>
			{/* Delete */}
			<button
				type="button"
				aria-label={`Remove ${character.Name || "character"}`}
				title="Remove from encounter"
				onClick={(e) => { e.stopPropagation(); onRemove(); }}
				onMouseEnter={() => setDeleteHovered(true)}
				onMouseLeave={() => setDeleteHovered(false)}
				style={{
					display: "flex",
					alignItems: "center",
					justifyContent: "center",
					width: 32,
					height: 32,
					borderRadius: "50%",
					border: "none",
					background: deleteHovered ? "rgba(211,47,47,0.1)" : "transparent",
					color: deleteHovered ? "#d32f2f" : "#bdbdbd",
					cursor: "pointer",
					transition: "background 0.2s, color 0.2s",
					padding: 0,
					marginLeft: 8,
					flexShrink: 0,
				}}
			>
				<svg aria-hidden="true" focusable="false" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
					<polyline points="3 6 5 6 21 6" />
					<path d="M19 6l-1 14H6L5 6" />
					<path d="M10 11v6" />
					<path d="M14 11v6" />
					<path d="M9 6V4h6v2" />
				</svg>
			</button>
			</div>
			{/* Conditions strip. Clicks here must not select the row, so stop
			    propagation on the whole strip. */}
			<div
				onClick={(e) => e.stopPropagation()}
				style={{
					display: "flex",
					alignItems: "center",
					flexWrap: "wrap",
					gap: 6,
					marginTop: 8,
					paddingTop: 8,
					borderTop: "1px dashed #e0e0e0",
				}}
			>
				<span
					style={{
						fontWeight: 500,
						color: "#1976d2",
						fontSize: "0.8rem",
					}}
				>
					Conditions:
				</span>
				{conditions.length === 0 && !addingCondition && (
					<span style={{ color: "#9e9e9e", fontSize: "0.8rem" }}>
						none
					</span>
				)}
				{conditions.map((cond) => {
					// For a leveled condition the chip's tooltip explains what that
					// level does (e.g. exhaustion 5 = speed 0); notes come first when
					// the DM wrote one.
					const levelEffect =
						cond.Level != null
							? conditionCatalog.find((i) => i.Name === cond.Condition)
									?.LevelEffects?.[cond.Level - 1]
							: undefined;
					const tooltip =
						[cond.Note, levelEffect].filter(Boolean).join(" — ") || undefined;
					return (
					<span
						key={cond.ID}
						title={tooltip}
						style={{
							display: "inline-flex",
							alignItems: "center",
							gap: 4,
							background: "#ede7f6",
							color: "#4527a0",
							borderRadius: 12,
							padding: "2px 8px",
							fontSize: "0.8rem",
							fontWeight: 600,
						}}
					>
						{cond.Condition}
						{cond.Level != null && <span>&nbsp;{cond.Level}</span>}
						{cond.DurationRounds != null && (
							<span style={{ fontWeight: 400 }}>
								({cond.DurationRounds})
							</span>
						)}
						<button
							type="button"
							aria-label={`Remove ${cond.Condition}`}
							onClick={() => onRemoveCondition(cond.ID)}
							style={{
								border: "none",
								background: "none",
								color: "#7e57c2",
								cursor: "pointer",
								padding: 0,
								lineHeight: 1,
								fontSize: "0.9rem",
							}}
						>
							✕
						</button>
					</span>
					);
				})}
				{addingCondition ? (
					<span
						style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
					>
						<select
							value={newCondition}
							onChange={(e) => handleConditionChange(e.target.value)}
							autoFocus
							style={{
								fontSize: "0.8rem",
								padding: "2px 4px",
								borderRadius: 4,
							}}
						>
							<option value="">Select…</option>
							{availableConditions.map((info) => (
								<option key={info.Name} value={info.Name}>
									{info.Name}
								</option>
							))}
						</select>
						{maxLevel > 0 && (
							<select
								value={newLevel}
								onChange={(e) => setNewLevel(e.target.value)}
								title="Exhaustion level"
								style={{
									fontSize: "0.8rem",
									padding: "2px 4px",
									borderRadius: 4,
								}}
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
							</select>
						)}
						<input
							type="number"
							min={1}
							value={newDuration}
							onChange={(e) => setNewDuration(e.target.value)}
							placeholder="∞"
							title="Duration in rounds (blank = until removed)"
							style={{
								width: 48,
								fontSize: "0.8rem",
								padding: "2px 4px",
								borderRadius: 4,
							}}
						/>
						<button
							type="button"
							onClick={submitCondition}
							disabled={!newCondition}
							style={{
								border: "none",
								background: "#1976d2",
								color: "#fff",
								borderRadius: 4,
								padding: "2px 8px",
								fontSize: "0.8rem",
								cursor: newCondition ? "pointer" : "not-allowed",
								opacity: newCondition ? 1 : 0.6,
							}}
						>
							Add
						</button>
						<button
							type="button"
							onClick={resetConditionForm}
							style={{
								border: "none",
								background: "none",
								color: "#757575",
								cursor: "pointer",
								fontSize: "0.8rem",
							}}
						>
							Cancel
						</button>
					</span>
				) : (
					availableConditions.length > 0 && (
						<button
							type="button"
							aria-label="Add condition"
							onClick={() => setAddingCondition(true)}
							style={{
								border: "1px dashed #90caf9",
								background: "none",
								color: "#1976d2",
								borderRadius: 12,
								padding: "1px 8px",
								fontSize: "0.8rem",
								cursor: "pointer",
							}}
						>
							+ Condition
						</button>
					)
				)}
			</div>
		</div>
	);
};
