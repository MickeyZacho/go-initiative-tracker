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
import {
	AddCustomCharacterControl,
	type CustomCharacterDraft,
} from "./AddCustomCharacterControl";
import { CombatLog } from "./CombatLog";
import { CombatBar } from "./CombatBar";
import type { SelectChangeEvent } from "@mui/material/Select";

import { useEncounters } from "../hooks/useEncounters";
import { useCharacters } from "../hooks/useCharacters";
import { useNpcTemplates } from "../hooks/useNpcTemplates";
import { useCombatLog } from "../hooks/useCombatLog";
import { useCombatState } from "../hooks/useCombatState";
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
	// True once the character has spent its reaction this round; reset by the
	// backend when the character's turn begins.
	ReactionUsed: boolean;
	Conditions?: Condition[];
}

// (Encounter and LedgerEntry interfaces removed; now provided by hooks or not needed)

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

	// Round counter
	const { round, fetchCombatState } = useCombatState();
	const [combatBusy, setCombatBusy] = useState(false);

	// Local state
	const [libraryCharacters, setLibraryCharacters] = useState<Character[]>([]);
	const [conditionCatalog, setConditionCatalog] = useState<ConditionInfo[]>([]);
	const [actionError, setActionError] = useState<string>("");
	const [selectedAddCharacterId, setSelectedAddCharacterId] =
		useState<number>(0);
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

	// Round is per encounter; load it whenever the selected encounter changes.
	useEffect(() => {
		void fetchCombatState(encounterId);
	}, [encounterId, fetchCombatState]);

	// Live sync: when any viewer changes this encounter, the backend pushes an
	// event and we re-pull characters, ledger and round so every viewer stays in
	// step.
	useEncounterEvents(encounterId, () => {
		if (!encounterId) return;
		void fetchCharacters(encounterId);
		void fetchLedger(encounterId);
		void fetchCombatState(encounterId);
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

	// Hand the turn to a specific character (the row's "give turn" button), so
	// the server's turn pointer moves and a subsequent "next turn" continues from
	// there. The row updates IsActive locally for instant feedback; this syncs
	// the server and re-pulls on failure.
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
					"Failed to give turn",
				);
			} catch (err) {
				setActionError(
					err instanceof Error ? err.message : "Failed to give turn",
				);
				await fetchCharacters(encounterId);
			}
		},
		[encounterId, fetchCharacters],
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

	// Toggle a character's reaction between spent and available. We flip it locally
	// first for instant feedback, then persist; the live-sync refetch (and the
	// automatic reset on a character's turn) keep every viewer in step.
	const setReaction = useCallback(
		async (characterID: number, used: boolean) => {
			if (!encounterId || !characterID) return;
			setCharacters((prev) =>
				prev.map((c) =>
					c.ID === characterID ? { ...c, ReactionUsed: used } : c,
				),
			);
			try {
				await apiPost(
					"/encounters/combat/reaction",
					{
						encounter_id: encounterId,
						character_id: characterID,
						reaction_used: used,
					},
					"Failed to update reaction",
				);
			} catch (err) {
				setActionError(
					err instanceof Error ? err.message : "Failed to update reaction",
				);
				await fetchCharacters(encounterId);
			}
		},
		[encounterId, setCharacters, fetchCharacters],
	);

	// Removing drops the creature's HP, initiative and conditions for this
	// encounter, and the trash icon sits among the other row controls, so ask
	// first — the other pages already confirm their deletes.
	const removeCharacter = async (character: Character) => {
		const name = character.Name || "this character";
		if (!window.confirm(`Remove ${name} from the encounter?`)) {
			return;
		}
		setActionError("");
		try {
			await apiPost(
				"/remove-character-from-encounter",
				{
					encounter_id: encounterId,
					character_id: character.ID,
				},
				"Failed to remove character",
			);
			await fetchCharacters(encounterId);
		} catch (err) {
			setActionError(
				err instanceof Error ? err.message : "Failed to remove character",
			);
		}
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

	// Create a brand-new character/NPC and drop it straight into the encounter.
	// /save-character with id=0 + encounter_id both persists it and adds it, so we
	// also refresh the library list to pick up the newly created character.
	const addCustomCharacterToEncounter = async (draft: CustomCharacterDraft) => {
		if (!encounterId) {
			return;
		}
		setActionError("");
		try {
			await apiPost(
				"/save-character",
				{
					id: 0,
					name: draft.name,
					armorClass: draft.armorClass,
					maxHP: draft.maxHP,
					currentHP: draft.maxHP,
					initiative: draft.initiative,
					toHitModifier: draft.toHitModifier,
					isActive: false,
					type: draft.type,
					encounter_id: encounterId,
				},
				"Failed to add custom character",
			);
			await fetchCharacters(encounterId);
			await fetchLibraryCharacters();
		} catch (err) {
			setActionError(
				err instanceof Error
					? err.message
					: "Failed to add custom character",
			);
		}
	};

	// Run one combat-control call (start / next / end), then refresh the rows,
	// round and log. combatBusy guards against double-advancing on a fast
	// double-click or held spacebar.
	const runCombatAction = useCallback(
		async (path: string, failure: string) => {
			if (!encounterId || combatBusy) return;
			setActionError("");
			setCombatBusy(true);
			try {
				await apiPost(path, { encounter_id: encounterId }, failure);
				await Promise.all([
					fetchCharacters(encounterId),
					fetchCombatState(encounterId),
					fetchLedger(encounterId),
				]);
			} catch (err) {
				setActionError(err instanceof Error ? err.message : failure);
			} finally {
				setCombatBusy(false);
			}
		},
		[encounterId, combatBusy, fetchCharacters, fetchCombatState, fetchLedger],
	);

	const nextCharacter = useCallback(() => {
		if (characters.length === 0) return;
		void runCombatAction("/encounters/combat/next-turn", "Failed to advance turn");
	}, [characters.length, runCombatAction]);

	const startCombat = () =>
		void runCombatAction("/encounters/combat/start", "Failed to start combat");

	const endCombat = () => {
		if (
			!window.confirm(
				"End combat? This clears whose turn it is and resets the round counter. HP and conditions are kept.",
			)
		) {
			return;
		}
		void runCombatAction("/encounters/combat/setup", "Failed to end combat");
	};

	const activeCharacter = characters.find((c) => c.IsActive) ?? null;

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

	// Spacebar advances the turn (the Next button is the visible equivalent).
	// Ignore it while the user is typing or has a control focused, where Space
	// already means something, and ignore auto-repeat from a held key.
	useEffect(() => {
		const handleKeyDown = (e: KeyboardEvent) => {
			if (e.code !== "Space" || e.repeat) return;
			const t = e.target;
			if (
				t instanceof HTMLInputElement ||
				t instanceof HTMLTextAreaElement ||
				t instanceof HTMLSelectElement ||
				t instanceof HTMLButtonElement ||
				(t instanceof HTMLElement &&
					(t.isContentEditable || t.getAttribute("role") === "button"))
			) {
				return;
			}
			e.preventDefault();
			nextCharacter();
		};
		window.addEventListener("keydown", handleKeyDown);
		return () => window.removeEventListener("keydown", handleKeyDown);
	}, [nextCharacter]);

	return (
		<Box display="flex" justifyContent="center" alignItems="center">
			<Card
				sx={{
					width: "100%",
					// Card defaults to overflow:hidden, which would stop the combat
					// bar from sticking to the viewport.
					overflow: "visible",
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
								sx={{ fontWeight: 500, color: "primary.main" }}
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
							{/* Same order as the backend's turn logic (initiative DESC,
							    character_id ASC) so ties display in the order they act. */}
							{[...characters]
								.sort((a, b) => b.Initiative - a.Initiative || a.ID - b.ID)
								.map((character) => (
									<div key={character.ID} style={{ width: "100%" }}>
										<CharacterRow
											character={character}
											setCharacters={setCharacters}
											onGiveTurn={setActiveCharacter}
											onSave={saveCharacter}
											onRemove={() => removeCharacter(character)}
											conditionCatalog={conditionCatalog}
											onAddCondition={addCondition}
											onRemoveCondition={removeCondition}
											onSetReaction={setReaction}
										/>
									</div>
								))}
						</Stack>
						{encounterId > 0 && (
							<CombatBar
								round={round}
								activeName={activeCharacter?.Name ?? null}
								hasCombatants={characters.length > 0}
								busy={combatBusy}
								onStart={startCombat}
								onNext={nextCharacter}
								onReset={endCombat}
							/>
						)}
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
									<Box>
										<Typography
											variant="subtitle2"
											color="text.secondary"
											gutterBottom
										>
											Add Custom Character
										</Typography>
										<AddCustomCharacterControl
											addCustomCharacterToEncounter={
												addCustomCharacterToEncounter
											}
											encounterId={encounterId}
										/>
									</Box>
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
