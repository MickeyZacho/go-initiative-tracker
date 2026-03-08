import { CharacterList } from "./components/CharacterList";
import AuthButtons from "./components/AuthButtons";
import CharactersPage from "./components/CharactersPage";
import { Button, Stack } from "@mui/material";
import { useState } from "react";

function App() {
	const [view, setView] = useState<"combat" | "characters">("characters");

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
					variant={view === "combat" ? "contained" : "outlined"}
					onClick={() => setView("combat")}
				>
					Combat
				</Button>
			</Stack>
			{view === "characters" ? <CharactersPage /> : <CharacterList />}
		</div>
	);
}

export default App;
