import { useState, useEffect, useCallback } from "react";
import { CharacterRow } from "./CharacterRow";
import {
	Card,
	CardContent,
	Typography,
	Box,
	Select,
	MenuItem,
	FormControl,
	InputLabel,
	Stack,
} from "@mui/material";
import { AddCharacterControl } from "./AddCharacterControl";
import { AddNpcControl } from "./AddNpcControl";
import { CombatControls } from "./CombatControls";
import { CombatLog } from "./CombatLog";
import type { SelectChangeEvent } from "@mui/material/Select";

import { useEncounters } from "../hooks/useEncounters";
import { useCharacters } from "../hooks/useCharacters";
import { useNpcTemplates } from "../hooks/useNpcTemplates";
import { useCombatLog } from "../hooks/useCombatLog";

export interface Character {
	ID: number;
	Name: string;
	ArmorClass: number;
	ToHitModifier?: number;
	MaxHP: number;
	CurrentHP: number;
	Initiative: number;
	IsActive: boolean;
	OwnerID: string;
}

// (Encounter and LedgerEntry interfaces removed; now provided by hooks or not needed)

interface QuickActionInput {
	targetId: number;
	amount: string;
}

interface CharacterListProps {
	initialEncounterId?: number | null;
}

export const CharacterList: React.FC<CharacterListProps> = ({
	initialEncounterId,
}) => {
	// Encounters
	const {
		encounters,
		encounterId,
		setEncounterId,
		fetchEncounters,
		error: encountersError,
		isLoading: encountersLoading,
	} = useEncounters(initialEncounterId);

	// Characters
	const {
		characters,
		setCharacters,
		fetchCharacters,
		isLoading: charactersLoading,
		error: charactersError,
	} = useCharacters();

	// NPC Templates
	const {
		npcTemplates,
		selectedAddNpcId,
		setSelectedAddNpcId,
		fetchNpcTemplates,
	} = useNpcTemplates();

	// Combat Log
	const { ledgerEntries, fetchLedger } = useCombatLog();

	// Local state
	const [libraryCharacters, setLibraryCharacters] = useState<Character[]>([]);
	const [actionError, setActionError] = useState<string>("");
	const [selectedAddCharacterId, setSelectedAddCharacterId] =
		useState<number>(0);
	const [logActorId, setLogActorId] = useState<number>(0);
	const [logTargetId, setLogTargetId] = useState<number>(0);
	const [logActionType, setLogActionType] = useState<string>("note");
	const [logHPChange, setLogHPChange] = useState<string>("0");
	const [logDescription, setLogDescription] = useState<string>("");
	const [quickActionByActor, setQuickActionByActor] = useState<
		Record<number, QuickActionInput>
	>({});
	const [, setSelected] = useState<number | null>(null);

	// Compose error and loading states from hooks
	const isLoading = encountersLoading || charactersLoading;
	const composedError = encountersError || charactersError;
	// Fetch library characters and initialize on mount
	const fetchLibraryCharacters = useCallback(async () => {
		try {
			const response = await fetch("/api/characters/library", {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to fetch character library");
			}
			const payload = await response.json();
			const data: Character[] = Array.isArray(payload) ? payload : [];
			setLibraryCharacters(data);
			if (data.length > 0) {
				setSelectedAddCharacterId(data[0].ID);
			}
		} catch {
			setLibraryCharacters([]);
		}
	}, []);

	useEffect(() => {
		fetchEncounters(fetchCharacters, fetchLedger);
		fetchLibraryCharacters();
		fetchNpcTemplates();
	}, [
		fetchEncounters,
		fetchCharacters,
		fetchLedger,
		fetchLibraryCharacters,
		fetchNpcTemplates,
	]);
	// Add NPC to encounter
	const addNpcToEncounter = async () => {
		if (!encounterId || !selectedAddNpcId) {
			return;
		}
		setActionError("");
		try {
			const response = await fetch(
				"/api/npcs/templates/create-character",
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						encounter_id: encounterId,
						npc_template_id: selectedAddNpcId,
					}),
				},
			);
			const data = await response.json();
			if (!response.ok || data.status !== "success") {
				throw new Error(
					data.message || "Failed to add NPC to encounter",
				);
			}
			await fetchCharacters(encounterId);
		} catch (err) {
			setActionError(
				err instanceof Error ? err.message : "Failed to add NPC",
			);
		}
	};

	useEffect(() => {
		const available = libraryCharacters.filter(
			(libChar) =>
				!characters.some((encChar) => encChar.ID === libChar.ID),
		);
		if (available.length === 0) {
			setSelectedAddCharacterId(0);
			return;
		}
		if (!available.some((c) => c.ID === selectedAddCharacterId)) {
			setSelectedAddCharacterId(available[0].ID);
		}
	}, [libraryCharacters, characters, selectedAddCharacterId]);

	const handleEncounterChange = async (event: SelectChangeEvent) => {
		const newId = Number(event.target.value);
		setEncounterId(newId);
		await fetchCharacters(newId);
		await fetchLedger(newId);
	};

	const addLogEntry = async () => {
		if (!encounterId || !logActorId) {
			setActionError(
				"Select an encounter and actor before adding a log entry",
			);
			return;
		}
		setActionError("");
		try {
			await createLedgerEntry(
				logActorId,
				logTargetId,
				logActionType,
				Number(logHPChange) || 0,
				logDescription.trim(),
			);
			setLogDescription("");
			setLogHPChange("0");
			await fetchLedger(encounterId);
		} catch (err) {
			setActionError(
				err instanceof Error ? err.message : "Failed to add log entry",
			);
		}
	};

	const createLedgerEntry = useCallback(
		async (
			actorID: number,
			targetID: number,
			actionType: string,
			hpChange: number,
			description: string,
		) => {
			const response = await fetch("/api/encounters/ledger/add", {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					encounter_id: encounterId,
					actor_id: actorID,
					target_id: targetID,
					action_type: actionType,
					hp_change: hpChange,
					description,
				}),
			});
			const payload = await response.json();
			if (!response.ok || payload.status !== "success") {
				throw new Error(
					payload.message || "Failed to add combat log entry",
				);
			}
		},
		[encounterId],
	);

	const handleQuickActionChange = useCallback(
		(
			actorID: number,
			field: keyof QuickActionInput,
			value: number | string,
		) => {
			setQuickActionByActor((prev) => ({
				...prev,
				[actorID]: {
					targetId: prev[actorID]?.targetId ?? 0,
					amount: prev[actorID]?.amount ?? "1",
					[field]: value,
				},
			}));
		},
		[],
	);

	const applyQuickAction = useCallback(
		async (actor: Character, actionType: "attack" | "heal") => {
			if (!encounterId) {
				return;
			}
			const config = quickActionByActor[actor.ID];
			const targetID = config?.targetId ?? 0;
			const amount = Math.floor(Number(config?.amount ?? "0"));
			if (!targetID || amount <= 0) {
				setActionError(
					"Select a target and enter an amount greater than 0",
				);
				return;
			}
			const target = characters.find((c) => c.ID === targetID);
			if (!target) {
				setActionError("Target character not found");
				return;
			}

			const newHP =
				actionType === "attack"
					? Math.max(0, target.CurrentHP - amount)
					: Math.min(target.MaxHP, target.CurrentHP + amount);
			const hpChange = newHP - target.CurrentHP;

			setActionError("");
			try {
				await saveCharacter({ ...target, CurrentHP: newHP });
				await createLedgerEntry(
					actor.ID,
					target.ID,
					actionType,
					hpChange,
					`${actor.Name} ${actionType === "attack" ? "attacks" : "heals"} ${target.Name}`,
				);
				await fetchCharacters(encounterId);
				await fetchLedger(encounterId);
			} catch (err) {
				setActionError(
					err instanceof Error ? err.message : "Action failed",
				);
			}
		},
		[
			characters,
			createLedgerEntry,
			encounterId,
			fetchCharacters,
			fetchLedger,
			quickActionByActor,
		],
	);

	const handleQuickAmountKeyDown = useCallback(
		(event: React.KeyboardEvent<HTMLElement>, actor: Character) => {
			if (event.key !== "Enter") {
				return;
			}
			event.preventDefault();
			if (event.shiftKey) {
				void applyQuickAction(actor, "heal");
				return;
			}
			void applyQuickAction(actor, "attack");
		},
		[applyQuickAction],
	);

	const saveCharacter = async (character: Character) => {
		const idToSend = character.ID > 2147483647 ? 0 : character.ID;
		const response = await fetch("/api/save-character", {
			method: "POST",
			credentials: "include",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				id: idToSend,
				name: character.Name,
				armorClass: Number(character.ArmorClass),
				maxHP: Number(character.MaxHP),
				currentHP: Number(character.CurrentHP),
				initiative: Number(character.Initiative),
				toHitModifier: Number(character.ToHitModifier ?? 0),
				isActive: Boolean(character.IsActive),
			}),
		});
		if (!response.ok) {
			throw new Error("Failed to save character");
		}
		await fetchCharacters(encounterId);
	};

	const removeCharacter = async (characterID: number) => {
		const response = await fetch("/api/remove-character-from-encounter", {
			method: "POST",
			credentials: "include",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				encounter_id: encounterId,
				character_id: characterID,
			}),
		});
		const data = await response.json();
		if (!response.ok || data.status !== "success") {
			throw new Error(data.message || "Failed to remove character");
		}
		await fetchCharacters(encounterId);
	};

	const addExistingCharacterToEncounter = async () => {
		if (!encounterId || !selectedAddCharacterId) {
			return;
		}
		try {
			const response = await fetch("/api/add-character-to-encounter", {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					encounter_id: encounterId,
					character_id: selectedAddCharacterId,
				}),
			});
			const data = await response.json();
			if (!response.ok || data.status !== "success") {
				throw new Error(
					data.message || "Failed to add character to encounter",
				);
			}
			await fetchCharacters(encounterId);
		} catch (err) {
			setActionError(
				err instanceof Error ? err.message : "Failed to add character",
			);
		}
	};

	const nextCharacter = useCallback(async () => {
		if (!encounterId || characters.length === 0) return;
		setActionError("");
		try {
			const response = await fetch("/api/encounters/combat/next-turn", {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ encounter_id: encounterId }),
			});
			const data = await response.json();
			if (!response.ok || data.status !== "success") {
				throw new Error(data.message || "Failed to advance turn");
			}
			await fetchCharacters(encounterId);
		} catch (err) {
			setActionError(
				err instanceof Error ? err.message : "Failed to advance turn",
			);
		}
	}, [encounterId, characters.length, fetchCharacters]);

	const availableLibraryCharacters = libraryCharacters.filter(
		(libChar) => !characters.some((encChar) => encChar.ID === libChar.ID),
	);
	const characterNameByID = useCallback(
		(id: number) => characters.find((c) => c.ID === id)?.Name || "Unknown",
		[characters],
	);
	const formatLogTime = useCallback((timestamp: string) => {
		const date = new Date(timestamp);
		if (Number.isNaN(date.getTime())) {
			return timestamp;
		}
		return date.toLocaleTimeString([], {
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
		});
	}, []);

	useEffect(() => {
		if (characters.length === 0) {
			setLogActorId(0);
			setLogTargetId(0);
			return;
		}
		if (!characters.some((c) => c.ID === logActorId)) {
			setLogActorId(characters[0].ID);
		}
		if (
			logTargetId !== 0 &&
			!characters.some((c) => c.ID === logTargetId)
		) {
			setLogTargetId(0);
		}
	}, [characters, logActorId, logTargetId]);

	useEffect(() => {
		setQuickActionByActor((prev) => {
			const next: Record<number, QuickActionInput> = {};
			for (const actor of characters) {
				const existing = prev[actor.ID];
				const fallbackTarget =
					characters.find((c) => c.ID !== actor.ID)?.ID ?? actor.ID;
				const validTarget =
					existing &&
					characters.some((c) => c.ID === existing.targetId)
						? existing.targetId
						: fallbackTarget;
				next[actor.ID] = {
					targetId: validTarget,
					amount: existing?.amount ?? "1",
				};
			}
			return next;
		});
	}, [characters]);

	// Spacebar triggers nextCharacter
	useEffect(() => {
		const handleKeyDown = (e: KeyboardEvent) => {
			if (
				e.code === "Space" &&
				!(
					e.target instanceof HTMLInputElement ||
					e.target instanceof HTMLTextAreaElement
				)
			) {
				e.preventDefault();
				nextCharacter();
			}
		};
		window.addEventListener("keydown", handleKeyDown);
		return () => window.removeEventListener("keydown", handleKeyDown);
	}, [nextCharacter, characters, encounterId]);

	return (
		<Box display="flex" justifyContent="center" alignItems="center">
			<Card
				sx={{
					width: 700,
					borderRadius: 3,
					boxShadow: 3,
					p: 3,
				}}
			>
				<CardContent>
					<Stack spacing={3}>
						{/* Encounter Switcher */}
						<FormControl
							variant="outlined"
							sx={{ minWidth: 220 }}
						>
							<InputLabel id="encounter-label">
								Encounter
							</InputLabel>
							<Select
								labelId="encounter-label"
								value={String(encounterId)}
								label="Encounter"
								onChange={handleEncounterChange}
								sx={{ fontWeight: 500, color: "#1976d2" }}
							>
								{encounters.map((enc) => (
									<MenuItem
										key={enc.ID}
										value={String(enc.ID)}
									>
										{enc.Name}
									</MenuItem>
								))}
							</Select>
						</FormControl>

						{composedError && (
							<Typography color="error">
								{composedError}
							</Typography>
						)}
						{actionError && (
							<Typography color="error">{actionError}</Typography>
						)}
						<Typography
							variant="h4"
							color="primary"
							fontWeight={700}
							letterSpacing={1}
						>
							Characters
						</Typography>
						{/* Character Rows */}
						<Stack spacing={2}>
							{[...characters]
								.sort((a, b) => b.Initiative - a.Initiative)
								.map((character) => (
									<div key={character.ID} style={{ width: "100%" }}>
										<CharacterRow
											character={character}
											setCharacters={setCharacters}
											setSelected={setSelected}
											onSave={saveCharacter}
											onRemove={() => removeCharacter(character.ID)}
										/>
									</div>
								))}
							<CombatControls
								characters={characters}
								quickActionByActor={quickActionByActor}
								handleQuickActionChange={handleQuickActionChange}
								handleQuickAmountKeyDown={handleQuickAmountKeyDown}
								applyQuickAction={applyQuickAction}
							/>
						</Stack>
						{/* Add Existing Character */}
						<Stack
							direction="row"
							spacing={2}
							justifyContent="center"
							alignItems="center"
							mt={2}
						>
							<AddCharacterControl
								availableLibraryCharacters={availableLibraryCharacters}
								selectedAddCharacterId={selectedAddCharacterId}
								setSelectedAddCharacterId={setSelectedAddCharacterId}
								addExistingCharacterToEncounter={addExistingCharacterToEncounter}
								encounterId={encounterId}
							/>
						</Stack>
						{/* Add NPC */}
						<Stack
							direction="row"
							spacing={2}
							justifyContent="center"
							alignItems="center"
							mt={1}
						>
							<AddNpcControl
								npcTemplates={npcTemplates}
								selectedAddNpcId={selectedAddNpcId}
								setSelectedAddNpcId={setSelectedAddNpcId}
								addNpcToEncounter={addNpcToEncounter}
								encounterId={encounterId}
							/>
						</Stack>
						{isLoading && (
							<Typography color="text.secondary">
								Loading...
							</Typography>
						)}
						<CombatLog
							ledgerEntries={ledgerEntries}
							characters={characters}
							logActorId={logActorId}
							setLogActorId={setLogActorId}
							logTargetId={logTargetId}
							setLogTargetId={setLogTargetId}
							logActionType={logActionType}
							setLogActionType={setLogActionType}
							logHPChange={logHPChange}
							setLogHPChange={setLogHPChange}
							logDescription={logDescription}
							setLogDescription={setLogDescription}
							addLogEntry={addLogEntry}
							encounterId={encounterId}
							formatLogTime={formatLogTime}
							characterNameByID={characterNameByID}
						/>
					</Stack>
				</CardContent>
			</Card>
		</Box>
	);
};
