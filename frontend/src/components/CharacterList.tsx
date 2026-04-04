import { useState, useEffect, useCallback } from "react";
import { CharacterRow } from "./CharacterRow";
import { parseJsonResponse } from "../lib/http";
import { apiUrl } from "../lib/api";
import {
	Card,
	CardContent,
	Typography,
	Box,
	Button,
	Select,
	MenuItem,
	FormControl,
	InputLabel,
	Stack,
	TextField,
} from "@mui/material";
import type { SelectChangeEvent } from "@mui/material/Select";

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

interface Encounter {
	ID: number;
	Name: string;
}

interface LedgerEntry {
	id: number;
	encounter_id: number;
	actor_id: number;
	actor_name: string;
	target_id: number;
	target_name: string;
	action_type: string;
	hp_change: number;
	description: string;
	created_at: string;
}

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
	const [encounters, setEncounters] = useState<Encounter[]>([]);
	const [encounterId, setEncounterId] = useState<number>(0);
	const [characters, setCharacters] = useState<Character[]>([]);
	const [libraryCharacters, setLibraryCharacters] = useState<Character[]>([]);
	const [selectedAddCharacterId, setSelectedAddCharacterId] =
		useState<number>(0);
	const [ledgerEntries, setLedgerEntries] = useState<LedgerEntry[]>([]);
	const [logActorId, setLogActorId] = useState<number>(0);
	const [logTargetId, setLogTargetId] = useState<number>(0);
	const [logActionType, setLogActionType] = useState<string>("note");
	const [logHPChange, setLogHPChange] = useState<string>("0");
	const [logDescription, setLogDescription] = useState<string>("");
	const [quickActionByActor, setQuickActionByActor] = useState<
		Record<number, QuickActionInput>
	>({});
	const [, setSelected] = useState<number | null>(null);
	const [isLoading, setIsLoading] = useState<boolean>(false);
	const [error, setError] = useState<string>("");
	const combatStarted = characters.some((c) => c.IsActive);
	const activeCharacter = characters.find((c) => c.IsActive) ?? null;

	const fetchLedger = useCallback(async (encId: number) => {
		if (!encId) {
			setLedgerEntries([]);
			return;
		}
		try {
			const response = await fetch(
				apiUrl(`/api/encounters/ledger?encounter_id=${encId}`),
				{ credentials: "include" },
			);
			const payload = await parseJsonResponse<{
				status?: string;
				entries?: LedgerEntry[];
				message?: string;
			}>(response);
			if (!response.ok || payload.status !== "success") {
				throw new Error(payload.message || "Failed to load combat log");
			}
			setLedgerEntries(
				Array.isArray(payload.entries) ? payload.entries : [],
			);
		} catch {
			setLedgerEntries([]);
		}
	}, []);

	const fetchCharacters = useCallback(async (encId: number) => {
		setIsLoading(true);
		setError("");
		try {
			await fetch(apiUrl("/api/select-encounter"), {
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ id: encId }),
			});
			const response = await fetch(
				apiUrl(`/api/characters?encounter_id=${encId}`),
				{ credentials: "include" },
			);
			if (!response.ok) {
				throw new Error("Failed to fetch characters");
			}
			const payload = await parseJsonResponse<unknown>(response);
			const data: Character[] = Array.isArray(payload) ? payload : [];
			setCharacters(data);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to fetch characters",
			);
			setCharacters([]);
		} finally {
			setIsLoading(false);
		}
	}, []);

	const fetchEncounters = useCallback(async () => {
		setIsLoading(true);
		setError("");
		try {
			const response = await fetch(apiUrl("/api/encounters"), {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to fetch encounters");
			}
			const payload = await parseJsonResponse<unknown>(response);
			const data: Encounter[] = Array.isArray(payload) ? payload : [];
			setEncounters(data);
			if (data.length > 0) {
				const preferredEncounterId =
					initialEncounterId &&
					data.some((enc) => enc.ID === initialEncounterId)
						? initialEncounterId
						: data[0].ID;
				setEncounterId(preferredEncounterId);
				await fetchCharacters(preferredEncounterId);
				await fetchLedger(preferredEncounterId);
			} else {
				setEncounterId(0);
				setCharacters([]);
				setLedgerEntries([]);
			}
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to fetch encounters",
			);
			setEncounters([]);
			setCharacters([]);
		} finally {
			setIsLoading(false);
		}
	}, [fetchCharacters, fetchLedger, initialEncounterId]);

	const fetchLibraryCharacters = useCallback(async () => {
		try {
			const response = await fetch(apiUrl("/api/characters/library"), {
				credentials: "include",
			});
			if (!response.ok) {
				throw new Error("Failed to fetch character library");
			}
			const payload = await parseJsonResponse<unknown>(response);
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
		fetchEncounters();
		fetchLibraryCharacters();
	}, [fetchEncounters, fetchLibraryCharacters]);

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
			setError("Select an actor before adding a combat log entry");
			return;
		}
		setError("");
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
			setError(
				err instanceof Error
					? err.message
					: "Failed to add combat log entry",
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
			const response = await fetch(apiUrl("/api/encounters/ledger/add"), {
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
			const payload = await parseJsonResponse<{
				status?: string;
				message?: string;
			}>(response);
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
				setError("Select a target and enter an amount greater than 0");
				return;
			}
			const target = characters.find((c) => c.ID === targetID);
			if (!target) {
				setError("Target not found");
				return;
			}

			const newHP =
				actionType === "attack"
					? Math.max(0, target.CurrentHP - amount)
					: Math.min(target.MaxHP, target.CurrentHP + amount);
			const hpChange = newHP - target.CurrentHP;

			setError("");
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
				setError(
					err instanceof Error
						? err.message
						: "Failed to apply combat action",
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

	const addCharacter = async () => {
		if (combatStarted || !encounterId || !selectedAddCharacterId) {
			return;
		}
		setError("");
		try {
			const response = await fetch(
				apiUrl("/add-character-to-encounter"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						encounter_id: encounterId,
						character_id: selectedAddCharacterId,
					}),
				},
			);
			const data = await parseJsonResponse<{
				status?: string;
				message?: string;
			}>(response);
			if (!response.ok || data.status !== "success") {
				throw new Error(
					data.message || "Failed to add character to encounter",
				);
			}
			await fetchCharacters(encounterId);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to add character to encounter",
			);
		}
	};

	const saveCharacter = async (character: Character) => {
		const idToSend = character.ID > 2147483647 ? 0 : character.ID;
		const response = await fetch(apiUrl("/save-character"), {
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
		const response = await fetch(
			apiUrl("/remove-character-from-encounter"),
			{
				method: "POST",
				credentials: "include",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({
					encounter_id: encounterId,
					character_id: characterID,
				}),
			},
		);
		const data = await parseJsonResponse<{
			status?: string;
			message?: string;
		}>(response);
		if (!response.ok || data.status !== "success") {
			throw new Error(data.message || "Failed to remove character");
		}
		await fetchCharacters(encounterId);
	};

	const addExistingCharacterToEncounter = async () => {
		if (combatStarted || !encounterId || !selectedAddCharacterId) {
			return;
		}
		setError("");
		try {
			const response = await fetch(
				apiUrl("/add-character-to-encounter"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({
						encounter_id: encounterId,
						character_id: selectedAddCharacterId,
					}),
				},
			);
			const data = await parseJsonResponse<{
				status?: string;
				message?: string;
			}>(response);
			if (!response.ok || data.status !== "success") {
				throw new Error(
					data.message || "Failed to add character to encounter",
				);
			}
			await fetchCharacters(encounterId);
		} catch (err) {
			setError(
				err instanceof Error
					? err.message
					: "Failed to add character to encounter",
			);
		}
	};

	const nextCharacter = useCallback(async () => {
		if (!encounterId || characters.length === 0) return;
		setError("");
		try {
			const response = await fetch(
				apiUrl("/api/encounters/combat/next-turn"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ encounter_id: encounterId }),
				},
			);
			const data = await parseJsonResponse<{
				status?: string;
				message?: string;
			}>(response);
			if (!response.ok || data.status !== "success") {
				throw new Error(data.message || "Failed to advance turn");
			}
			await fetchCharacters(encounterId);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to advance turn",
			);
		}
	}, [encounterId, characters.length, fetchCharacters]);

	const handleStartCombat = async () => {
		if (!encounterId || characters.length === 0) return;
		setError("");
		try {
			const response = await fetch(
				apiUrl("/api/encounters/combat/start"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ encounter_id: encounterId }),
				},
			);
			const data = await parseJsonResponse<{
				status?: string;
				message?: string;
			}>(response);
			if (!response.ok || data.status !== "success") {
				throw new Error(data.message || "Failed to start combat");
			}
			await fetchCharacters(encounterId);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to start combat",
			);
		}
	};

	const handleBackToSetup = async () => {
		if (!encounterId) return;
		setError("");
		try {
			const response = await fetch(
				apiUrl("/api/encounters/combat/setup"),
				{
					method: "POST",
					credentials: "include",
					headers: { "Content-Type": "application/json" },
					body: JSON.stringify({ encounter_id: encounterId }),
				},
			);
			const data = await parseJsonResponse<{
				status?: string;
				message?: string;
			}>(response);
			if (!response.ok || data.status !== "success") {
				throw new Error(data.message || "Failed to reset combat");
			}
			setSelected(null);
			await fetchCharacters(encounterId);
		} catch (err) {
			setError(
				err instanceof Error ? err.message : "Failed to reset combat",
			);
		}
	};

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
			if (!combatStarted) return;
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
	}, [nextCharacter, characters, encounterId, combatStarted]);

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
						{/* Encounter Switcher and Start/Back Button */}
						<Stack direction="row" spacing={2} alignItems="center">
							<FormControl
								variant="outlined"
								sx={{ minWidth: 220, flexGrow: 1 }}
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
							<Button
								variant={
									combatStarted ? "outlined" : "contained"
								}
								color={combatStarted ? "inherit" : "warning"}
								onClick={
									combatStarted
										? handleBackToSetup
										: handleStartCombat
								}
								disabled={
									!encounterId ||
									(!combatStarted && characters.length === 0)
								}
								sx={{ whiteSpace: "nowrap", minWidth: 140 }}
							>
								{combatStarted
									? "Back to Setup"
									: "Start Combat"}
							</Button>
						</Stack>

						{error && (
							<Typography color="error">{error}</Typography>
						)}
						<Typography
							variant="h4"
							color="primary"
							fontWeight={700}
							letterSpacing={1}
						>
							Characters
						</Typography>
						<Typography
							color={
								combatStarted
									? "success.main"
									: "text.secondary"
							}
						>
							Status: {combatStarted ? "In Combat" : "Setup"}
						</Typography>
						{combatStarted && activeCharacter && (
							<Typography color="primary" fontWeight={600}>
								Current Turn: {activeCharacter.Name}
							</Typography>
						)}
						<Stack spacing={2}>
							{[...characters]
								.sort((a, b) => b.Initiative - a.Initiative)
								.map((character) => {
									const quickConfig =
										quickActionByActor[character.ID];
									const quickTargetID =
										quickConfig?.targetId ?? 0;
									const quickAmountRaw =
										quickConfig?.amount ?? "1";
									const quickAmount = Math.floor(
										Number(quickAmountRaw),
									);
									const targetCharacter = characters.find(
										(c) => c.ID === quickTargetID,
									);
									const hasValidTarget =
										Boolean(targetCharacter);
									const hasValidAmount =
										Number.isFinite(quickAmount) &&
										quickAmount > 0;
									const attackPreviewHP = targetCharacter
										? Math.max(
												0,
												targetCharacter.CurrentHP -
													quickAmount,
											)
										: 0;
									const healPreviewHP = targetCharacter
										? Math.min(
												targetCharacter.MaxHP,
												targetCharacter.CurrentHP +
													quickAmount,
											)
										: 0;
									const rowValidationMessage = !hasValidTarget
										? "Pick a valid target"
										: !hasValidAmount
											? "Amount must be greater than 0"
											: "";

									return (
										<div
											style={{ width: "100%" }}
											key={character.ID}
										>
											<CharacterRow
												character={character}
												setCharacters={setCharacters}
												setSelected={setSelected}
											/>
											<Stack
												direction="row"
												spacing={1}
												useFlexGap
												flexWrap="wrap"
												alignItems="center"
												mt={1}
											>
												{!combatStarted && (
													<>
														<Button
															size="small"
															variant="contained"
															onClick={() =>
																saveCharacter(
																	character,
																)
															}
														>
															Save
														</Button>
														<Button
															size="small"
															color="error"
															variant="outlined"
															onClick={() =>
																removeCharacter(
																	character.ID,
																)
															}
														>
															Remove
														</Button>
													</>
												)}
												{combatStarted && (
													<>
														<FormControl
															size="small"
															sx={{
																minWidth: 140,
															}}
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
																	quickActionByActor[
																		character
																			.ID
																	]
																		?.targetId ??
																		0,
																)}
																onChange={(
																	event: SelectChangeEvent,
																) =>
																	handleQuickActionChange(
																		character.ID,
																		"targetId",
																		Number(
																			event
																				.target
																				.value,
																		),
																	)
																}
															>
																{characters.map(
																	(
																		targetChar,
																	) => (
																		<MenuItem
																			key={
																				targetChar.ID
																			}
																			value={String(
																				targetChar.ID,
																			)}
																		>
																			{
																				targetChar.Name
																			}
																		</MenuItem>
																	),
																)}
															</Select>
														</FormControl>
														<TextField
															size="small"
															type="number"
															label="Amount"
															value={
																quickAmountRaw
															}
															onChange={(event) =>
																handleQuickActionChange(
																	character.ID,
																	"amount",
																	event.target
																		.value,
																)
															}
															onKeyDown={(
																event,
															) =>
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
																applyQuickAction(
																	character,
																	"attack",
																)
															}
															disabled={
																!hasValidTarget ||
																!hasValidAmount
															}
														>
															Attack
														</Button>
														<Button
															size="small"
															color="success"
															variant="contained"
															onClick={() =>
																applyQuickAction(
																	character,
																	"heal",
																)
															}
															disabled={
																!hasValidTarget ||
																!hasValidAmount
															}
														>
															Heal
														</Button>
													</>
												)}
											</Stack>
											{combatStarted && (
												<Typography
													variant="caption"
													color={
														rowValidationMessage
															? "error"
															: "text.secondary"
													}
													sx={{
														display: "block",
														mt: 0.5,
													}}
												>
													{rowValidationMessage
														? rowValidationMessage
														: targetCharacter
															? `Preview: ${targetCharacter.Name} HP ${targetCharacter.CurrentHP} → ${attackPreviewHP} (Attack) / ${healPreviewHP} (Heal)  • Enter = Attack, Shift+Enter = Heal`
															: "Select a target to preview result"}
												</Typography>
											)}
										</div>
									);
								})}
						</Stack>
						<Stack
							direction="row"
							spacing={2}
							justifyContent="center"
							alignItems="center"
							mt={2}
						>
							{!combatStarted && (
								<>
									<FormControl sx={{ minWidth: 240 }}>
										<InputLabel id="add-existing-character-label">
											Add Existing Character
										</InputLabel>
										<Select
											labelId="add-existing-character-label"
											value={
												selectedAddCharacterId
													? String(
															selectedAddCharacterId,
														)
													: ""
											}
											label="Add Existing Character"
											onChange={(
												event: SelectChangeEvent,
											) =>
												setSelectedAddCharacterId(
													Number(event.target.value),
												)
											}
										>
											{availableLibraryCharacters.map(
												(libChar) => (
													<MenuItem
														key={libChar.ID}
														value={String(
															libChar.ID,
														)}
													>
														{libChar.Name}
													</MenuItem>
												),
											)}
										</Select>
									</FormControl>
									<Button
										variant="outlined"
										onClick={
											addExistingCharacterToEncounter
										}
										disabled={
											!encounterId ||
											!selectedAddCharacterId ||
											availableLibraryCharacters.length ===
												0
										}
									>
										Add to Encounter
									</Button>
								</>
							)}
							<Button
								variant="contained"
								color="success"
								onClick={addCharacter}
								disabled={combatStarted}
								sx={{
									fontWeight: 600,
									fontSize: "1rem",
									px: 3,
									py: 1,
								}}
							>
								Add Character
							</Button>
							<Button
								variant="contained"
								color="primary"
								onClick={nextCharacter}
								disabled={
									isLoading ||
									!combatStarted ||
									characters.length === 0
								}
								sx={{
									fontWeight: 600,
									fontSize: "1rem",
									px: 3,
									py: 1,
								}}
							>
								Next Character
							</Button>
						</Stack>
						{isLoading && (
							<Typography color="text.secondary">
								Loading...
							</Typography>
						)}
						<Stack spacing={1.5} mt={2}>
							<Typography variant="h6" fontWeight={700}>
								Combat Log
							</Typography>
							<Stack
								direction="row"
								spacing={1}
								useFlexGap
								flexWrap="wrap"
							>
								<FormControl sx={{ minWidth: 140 }}>
									<InputLabel id="log-actor-label">
										Actor
									</InputLabel>
									<Select
										labelId="log-actor-label"
										label="Actor"
										value={
											logActorId ? String(logActorId) : ""
										}
										onChange={(event: SelectChangeEvent) =>
											setLogActorId(
												Number(event.target.value),
											)
										}
									>
										{characters.map((character) => (
											<MenuItem
												key={character.ID}
												value={String(character.ID)}
											>
												{character.Name}
											</MenuItem>
										))}
									</Select>
								</FormControl>
								<FormControl sx={{ minWidth: 140 }}>
									<InputLabel id="log-target-label">
										Target
									</InputLabel>
									<Select
										labelId="log-target-label"
										label="Target"
										value={String(logTargetId)}
										onChange={(event: SelectChangeEvent) =>
											setLogTargetId(
												Number(event.target.value),
											)
										}
									>
										<MenuItem value="0">None</MenuItem>
										{characters.map((character) => (
											<MenuItem
												key={character.ID}
												value={String(character.ID)}
											>
												{character.Name}
											</MenuItem>
										))}
									</Select>
								</FormControl>
								<FormControl sx={{ minWidth: 140 }}>
									<InputLabel id="log-action-label">
										Action
									</InputLabel>
									<Select
										labelId="log-action-label"
										label="Action"
										value={logActionType}
										onChange={(event: SelectChangeEvent) =>
											setLogActionType(event.target.value)
										}
									>
										<MenuItem value="attack">
											Attack
										</MenuItem>
										<MenuItem value="heal">Heal</MenuItem>
										<MenuItem value="note">Note</MenuItem>
									</Select>
								</FormControl>
								<TextField
									label="HP Change"
									type="number"
									value={logHPChange}
									onChange={(event) =>
										setLogHPChange(event.target.value)
									}
									sx={{ width: 120 }}
								/>
								<TextField
									label="Description"
									value={logDescription}
									onChange={(event) =>
										setLogDescription(event.target.value)
									}
									sx={{ minWidth: 220, flex: 1 }}
								/>
								<Button
									variant="contained"
									onClick={addLogEntry}
									disabled={!encounterId || !logActorId}
								>
									Add Log
								</Button>
							</Stack>
							<Box
								sx={{
									maxHeight: 180,
									overflowY: "auto",
									border: 1,
									borderColor: "divider",
									borderRadius: 1,
									p: 1,
								}}
							>
								{ledgerEntries.length === 0 ? (
									<Typography color="text.secondary">
										No combat log entries yet.
									</Typography>
								) : (
									ledgerEntries.map((entry) => (
										<Typography
											key={entry.id}
											variant="body2"
											sx={{ mb: 0.5 }}
										>
											{`[${formatLogTime(entry.created_at)}] `}
											{entry.actor_name ||
												characterNameByID(
													entry.actor_id,
												)}
											{entry.target_id > 0
												? ` -> ${entry.target_name || characterNameByID(entry.target_id)}`
												: ""}
											{` [${entry.action_type}] ${entry.hp_change} ${entry.description}`}
										</Typography>
									))
								)}
							</Box>
						</Stack>
					</Stack>
				</CardContent>
			</Card>
		</Box>
	);
};
