import { CharacterList } from "./components/CharacterList";
import AuthButtons from "./components/AuthButtons";
import CharactersPage from "./components/CharactersPage";
import EncountersPage from "./components/EncountersPage";
import { Button, Stack } from "@mui/material";
import {
	Navigate,
	Route,
	Routes,
	useLocation,
	useNavigate,
	useParams,
} from "react-router-dom";
import NpcsPage from "./components/NpcsPage";
import FriendsPage from "./components/FriendsPage";

const NAV_ITEMS = [
	{ label: "Characters", path: "/characters" },
	{ label: "Encounters", path: "/encounters" },
	{ label: "Combat", path: "/combat" },
	{ label: "NPCs", path: "/npcs" },
	{ label: "Friends", path: "/friends" },
] as const;

// Reads the encounter id from the /combat/:encounterId route so the combat
// view can preselect it; /combat with no id falls back to the default.
function CombatRoute() {
	const { encounterId } = useParams();
	const parsed = encounterId ? Number(encounterId) : null;
	const initialEncounterId =
		parsed !== null && Number.isFinite(parsed) ? parsed : null;
	return <CharacterList initialEncounterId={initialEncounterId} />;
}

function App() {
	const navigate = useNavigate();
	const location = useLocation();

	return (
		<div
			style={{
				width: "100vw",
				height: "100vh",
				display: "flex",
				justifyContent: "center",
				alignItems: "flex-start",
			}}
		>
			<div style={{ padding: "2rem" }}>
				<AuthButtons />
				<h1>Initiative Tracker</h1>
				<Stack direction="row" spacing={1} mb={2}>
					{NAV_ITEMS.map((item) => (
						<Button
							key={item.path}
							variant={
								location.pathname.startsWith(item.path)
									? "contained"
									: "outlined"
							}
							onClick={() => navigate(item.path)}
						>
							{item.label}
						</Button>
					))}
				</Stack>
				<Routes>
					<Route
						path="/"
						element={<Navigate to="/characters" replace />}
					/>
					<Route path="/characters" element={<CharactersPage />} />
					<Route
						path="/encounters"
						element={
							<EncountersPage
								onOpenEncounter={(id) =>
									navigate(`/combat/${id}`)
								}
							/>
						}
					/>
					<Route path="/combat" element={<CombatRoute />} />
					<Route
						path="/combat/:encounterId"
						element={<CombatRoute />}
					/>
					<Route path="/npcs" element={<NpcsPage />} />
					<Route path="/friends" element={<FriendsPage />} />
					<Route
						path="*"
						element={<Navigate to="/characters" replace />}
					/>
				</Routes>
			</div>
		</div>
	);
}

export default App;
