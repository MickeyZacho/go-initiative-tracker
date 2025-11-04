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
}) => {
	const [editing, setEditing] = React.useState<{
		field: EditableFieldName | null;
		value: string;
	}>({ field: null, value: "" });

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
		setCharacters((prev) =>
			prev.map((c) =>
				c.ID === character.ID ? { ...c, [field]: newValue } : c
			)
		);
		setEditing({ field: null, value: "" });
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
		setCharacters((prev) =>
			prev.map((c) => ({ ...c, IsActive: c.ID === character.ID }))
		);
		setSelected(character.ID);
	};

	return (
		<div
			className={`character-row${character.IsActive ? " active" : ""}`}
			style={{
				display: "flex",
				alignItems: "center",
				borderRadius: 8,
				border:
					"2px solid " + (character.IsActive ? "#1976d2" : "#e0e0e0"),
				background: character.IsActive ? "#f0f7ff" : "#fff",
				boxShadow: character.IsActive
					? "0 2px 8px rgba(25,118,210,0.08)"
					: "none",
				transition: "background 0.2s, border-color 0.2s",
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
		</div>
	);
};
