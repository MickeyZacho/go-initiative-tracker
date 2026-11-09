import { CharacterList } from "./components/CharacterList";
import AuthButtons from "./components/AuthButtons";
import CharactersPage from "./components/CharactersPage";
import EncountersPage from "./components/EncountersPage";
import { AppBar, Box, Button, Container, Stack, Toolbar, Typography } from "@mui/material";
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
import Footer from "./components/Footer";
import ColorModeToggle from "./components/ColorModeToggle";

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
	// The combat table needs the room for its stat columns; the list pages read
	// better at a narrower measure.
	const maxWidth = location.pathname.startsWith("/combat") ? "lg" : "md";

	return (
		<Box
			sx={{
				minHeight: "100vh",
				width: "100%",
				display: "flex",
				flexDirection: "column",
			}}
		>
			<AppBar position="sticky" color="default" elevation={1}>
				<Container maxWidth="lg" disableGutters>
					<Toolbar
						disableGutters
						sx={{
							px: { xs: 2, sm: 3 },
							gap: 2,
							justifyContent: "space-between",
							flexWrap: "wrap",
						}}
					>
						<Typography
							variant="h6"
							component="h1"
							noWrap
							sx={{ fontWeight: 700 }}
						>
							Initiative Tracker
						</Typography>
						<Stack
							direction="row"
							spacing={1}
							sx={{ flexWrap: "wrap", rowGap: 1 }}
						>
							{NAV_ITEMS.map((item) => (
								<Button
									key={item.path}
									size="small"
									variant={
										location.pathname.startsWith(item.path)
											? "contained"
											: "text"
									}
									onClick={() => navigate(item.path)}
								>
									{item.label}
								</Button>
							))}
						</Stack>
						<Stack direction="row" spacing={1} alignItems="center">
							<ColorModeToggle />
							<AuthButtons />
						</Stack>
					</Toolbar>
				</Container>
			</AppBar>
			<Container
				maxWidth={maxWidth}
				sx={{
					py: 4,
					px: { xs: 2, sm: 3 },
					textAlign: "left",
					flexGrow: 1,
				}}
			>
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
			</Container>
			<Footer />
		</Box>
	);
}

export default App;
