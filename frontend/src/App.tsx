import { CharacterList } from "./components/CharacterList";
import AuthButtons from "./components/AuthButtons";
import CharactersPage from "./components/CharactersPage";
import EncountersPage from "./components/EncountersPage";
import { Button, Stack } from "@mui/material";
import { useState } from "react";

function App() {
	const [view, setView] = useState<"combat" | "characters" | "encounters">(
		"characters",
	);
	const [selectedEncounterId, setSelectedEncounterId] = useState<
		number | null
	>(null);

	const handleOpenEncounter = (encounterId: number) => {
		setSelectedEncounterId(encounterId);
		setView("combat");
	};

	return (
		<div style={{ padding: "2rem" }}>
			<AuthButtons />
			<h1>Initiative Tracker</h1>
			<Stack direction="row" spacing={1} mb={2}>
				<Button
					variant={view === "characters" ? "contained" : "outlined"}
					onClick={() => setView("characters")}
				>
					Characters
				</Button>
				<Button
					variant={view === "encounters" ? "contained" : "outlined"}
					onClick={() => setView("encounters")}
				>
					Encounters
				</Button>
				<Button
					variant={view === "combat" ? "contained" : "outlined"}
					onClick={() => setView("combat")}
				>
					Combat
				</Button>
			</Stack>
			{view === "characters" && <CharactersPage />}
			{view === "encounters" && (
				<EncountersPage onOpenEncounter={handleOpenEncounter} />
			)}
			{view === "combat" && (
				<CharacterList initialEncounterId={selectedEncounterId} />
			)}
		</div>
	);
}

export default App;
