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
	Accordion,
	AccordionSummary,
	AccordionDetails,
	Button,
	Collapse,
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
import { useEncounterEvents } from "../hooks/useEncounterEvents";
import { apiGetArray, apiPost } from "../lib/http";

export interface Condition {
	ID: number;
	EncounterID: number;
	CharacterID: number;
	Condition: string;
	DurationRounds: number | null;
	// Non-null only for leveled conditions (Exhaustion, 1-6).
	Level: number | null;
	Note: string;
}

// One catalog entry from the backend. MaxLevel is 0 for ordinary conditions and
// >0 for leveled ones (Exhaustion), which is what tells the picker to prompt for
// a level; LevelEffects describes each level, used as tooltip text.
export interface ConditionInfo {
	Name: string;
	MaxLevel: number;
	LevelEffects?: string[];
}

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
	Type: string;
	Conditions?: Condition[];
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
	const [conditionCatalog, setConditionCatalog] = useState<ConditionInfo[]>([]);
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
	// Secondary combat surfaces are collapsed by default so the initiative list
	// stays the focus; users open them on demand.
	const [showAddControls, setShowAddControls] = useState<boolean>(false);

	// Compose error and loading states from hooks
	const isLoading = encountersLoading || charactersLoading;
	const composedError = encountersError || charactersError;
	// Fetch library characters and initialize on mount
	const fetchLibraryCharacters = useCallback(async () => {
		try {
			const data = await apiGetArray<Character>("/characters/library");
			setLibraryCharacters(data);
			if (data.length > 0) {
				setSelectedAddCharacterId(data[0].ID);
			}
		} catch {
			setLibraryCharacters([]);
		}
	}, []);

	// The condition catalog is a static, backend-owned list (the 5e set); fetch it
	// once so the row picker stays in sync with what the server will accept.
	useEffect(() => {
		void apiGetArray<ConditionInfo>("/encounters/conditions/catalog")
			.then(setConditionCatalog)
			.catch(() => setConditionCatalog([]));
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

	// Live sync: when any viewer changes this encounter, the backend pushes an
	// event and we re-pull characters + ledger so every viewer stays in step.
	useEncounterEvents(encounterId, () => {
		if (!encounterId) return;
		void fetchCharacters(encounterId);
		void fetchLedger(encounterId);
	});
	// Add NPC to encounter
	const addNpcToEncounter = async () => {
		if (!encounterId || !selectedAddNpcId) {
			return;
		}
		setActionError("");
		try {
			await apiPost(
				"/npcs/templates/create-character",
				{
					encounter_id: encounterId,
					npc_template_id: selectedAddNpcId,
				},
				"Failed to add NPC to encounter",
			);
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
			await apiPost(
				"/encounters/ledger/add",
				{
					encounter_id: encounterId,
					actor_id: actorID,
					target_id: targetID,
					action_type: actionType,
					hp_change: hpChange,
					description,
				},
				"Failed to add combat log entry",
			);
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

	const saveCharacter = useCallback(
		async (character: Character) => {
			const idToSend = character.ID > 2147483647 ? 0 : character.ID;
			await apiPost(
				"/save-character",
				{
					id: idToSend,
					name: character.Name,
					armorClass: Number(character.ArmorClass),
					maxHP: Number(character.MaxHP),
					currentHP: Number(character.CurrentHP),
					initiative: Number(character.Initiative),
					toHitModifier: Number(character.ToHitModifier ?? 0),
					isActive: Boolean(character.IsActive),
					type: character.Type,
					encounter_id: encounterId,
				},
				"Failed to save character",
			);
			await fetchCharacters(encounterId);
		},
		[encounterId, fetchCharacters],
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
			saveCharacter,
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

	// Persist the clicked character as the active turn holder so the server's
	// turn pointer moves with the selection (a subsequent "next turn" continues
	// from here, not the previous active character). The row updates IsActive
	// locally for instant feedback; this just syncs the server.
	const setActiveCharacter = useCallback(
		async (characterID: number) => {
			if (!encounterId || !characterID) return;
			try {
				await apiPost(
					"/encounters/combat/set-active",
					{
						encounter_id: encounterId,
						character_id: characterID,
					},
					"Failed to select character",
				);
			} catch (err) {
				setActionError(
					err instanceof Error
						? err.message
						: "Failed to select character",
				);
			}
		},
		[encounterId],
	);

	const addCondition = useCallback(
		async (
			characterID: number,
			condition: string,
			durationRounds: number | null,
			level: number | null,
		) => {
			if (!encounterId || !characterID || !condition) return;
			setActionError("");
			try {
				await apiPost(
					"/encounters/conditions/add",
					{
						encounter_id: encounterId,
						character_id: characterID,
						condition,
						duration_rounds: durationRounds,
						level,
					},
					"Failed to add condition",
				);
				await fetchCharacters(encounterId);
			} catch (err) {
				setActionError(
					err instanceof Error ? err.message : "Failed to add condition",
				);
			}
		},
		[encounterId, fetchCharacters],
	);

	const removeCondition = useCallback(
		async (conditionID: number) => {
			if (!encounterId || !conditionID) return;
			setActionError("");
			try {
				await apiPost(
					"/encounters/conditions/remove",
					{
						encounter_id: encounterId,
						condition_id: conditionID,
					},
					"Failed to remove condition",
				);
				await fetchCharacters(encounterId);
			} catch (err) {
				setActionError(
					err instanceof Error ? err.message : "Failed to remove condition",
				);
			}
		},
		[encounterId, fetchCharacters],
	);

	const removeCharacter = async (characterID: number) => {
		await apiPost(
			"/remove-character-from-encounter",
			{
				encounter_id: encounterId,
				character_id: characterID,
			},
			"Failed to remove character",
		);
		await fetchCharacters(encounterId);
	};

	const addExistingCharacterToEncounter = async () => {
		if (!encounterId || !selectedAddCharacterId) {
			return;
		}
		try {
			await apiPost(
				"/add-character-to-encounter",
				{
					encounter_id: encounterId,
					character_id: selectedAddCharacterId,
				},
				"Failed to add character to encounter",
			);
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
			await apiPost(
				"/encounters/combat/next-turn",
				{ encounter_id: encounterId },
				"Failed to advance turn",
			);
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
					width: "100%",
					borderRadius: 3,
					boxShadow: 3,
					p: { xs: 1, sm: 3 },
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
											onSelect={setActiveCharacter}
											onSave={saveCharacter}
											onRemove={() => removeCharacter(character.ID)}
											conditionCatalog={conditionCatalog}
											onAddCondition={addCondition}
											onRemoveCondition={removeCondition}
										/>
									</div>
								))}
							{characters.length > 0 && (
								<Accordion
									disableGutters
									elevation={0}
									slotProps={{
										transition: { unmountOnExit: true },
									}}
									sx={{
										border: 1,
										borderColor: "divider",
										borderRadius: 2,
										"&:before": { display: "none" },
									}}
								>
									<AccordionSummary expandIcon={<span>▾</span>}>
										<Typography fontWeight={600}>
											Attack / Heal Actions
										</Typography>
									</AccordionSummary>
									<AccordionDetails>
										<CombatControls
											characters={characters}
											quickActionByActor={quickActionByActor}
											handleQuickActionChange={handleQuickActionChange}
											handleQuickAmountKeyDown={handleQuickAmountKeyDown}
											applyQuickAction={applyQuickAction}
										/>
									</AccordionDetails>
								</Accordion>
							)}
						</Stack>
						{/* Add existing characters / NPCs — tucked behind a single
						    unobtrusive toggle so it doesn't dominate the view. */}
						<Box>
							<Button
								size="small"
								variant="text"
								onClick={() => setShowAddControls((prev) => !prev)}
								sx={{ textTransform: "none" }}
							>
								{showAddControls ? "▾" : "▸"} Add characters & NPCs
							</Button>
							<Collapse in={showAddControls}>
								<Stack spacing={2} mt={1}>
									<Stack
										direction="row"
										spacing={2}
										alignItems="center"
										flexWrap="wrap"
										useFlexGap
									>
										<AddCharacterControl
											availableLibraryCharacters={availableLibraryCharacters}
											selectedAddCharacterId={selectedAddCharacterId}
											setSelectedAddCharacterId={setSelectedAddCharacterId}
											addExistingCharacterToEncounter={addExistingCharacterToEncounter}
											encounterId={encounterId}
										/>
									</Stack>
									<Stack
										direction="row"
										spacing={2}
										alignItems="center"
										flexWrap="wrap"
										useFlexGap
									>
										<AddNpcControl
											npcTemplates={npcTemplates}
											selectedAddNpcId={selectedAddNpcId}
											setSelectedAddNpcId={setSelectedAddNpcId}
											addNpcToEncounter={addNpcToEncounter}
											encounterId={encounterId}
										/>
									</Stack>
								</Stack>
							</Collapse>
						</Box>
						{isLoading && characters.length === 0 && (
							<Typography color="text.secondary">
								Loading...
							</Typography>
						)}
						<Accordion
							disableGutters
							elevation={0}
							slotProps={{ transition: { unmountOnExit: true } }}
							sx={{
								border: 1,
								borderColor: "divider",
								borderRadius: 2,
								"&:before": { display: "none" },
							}}
						>
							<AccordionSummary expandIcon={<span>▾</span>}>
								<Typography fontWeight={600}>
									Combat Log
									{ledgerEntries.length > 0
										? ` (${ledgerEntries.length})`
										: ""}
								</Typography>
							</AccordionSummary>
							<AccordionDetails>
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
							</AccordionDetails>
						</Accordion>
					</Stack>
				</CardContent>
			</Card>
		</Box>
	);
};
