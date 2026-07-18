import React from "react";
import type { Character } from "./CharacterList";

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
}) => {
	const [editing, setEditing] = React.useState<{
		field: EditableFieldName | null;
		value: string;
	}>({ field: null, value: "" });
	const [deleteHovered, setDeleteHovered] = React.useState(false);

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
				alignItems: "center",
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
	);
};
