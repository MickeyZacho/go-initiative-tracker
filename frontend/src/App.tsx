import AuthButtons from "./components/AuthButtons";
import { lazy, Suspense, useState } from "react";
import {
	AppBar,
	Box,
	Container,
	IconButton,
	LinearProgress,
	Menu,
	MenuItem,
	Stack,
	Tab,
	Tabs,
	Toolbar,
	Typography,
} from "@mui/material";
import MenuIcon from "@mui/icons-material/Menu";
import {
	Navigate,
	Route,
	Routes,
	useLocation,
	useNavigate,
	useParams,
} from "react-router-dom";
import Footer from "./components/Footer";
import ColorModeToggle from "./components/ColorModeToggle";

// Each page is its own chunk, loaded on first visit, so the entry bundle is
// only the shell (React, MUI core, router, app bar). Keep these imports out of
// other modules, or the page gets pulled back into the main chunk.
const CharacterList = lazy(() =>
	import("./components/CharacterList").then((m) => ({
		default: m.CharacterList,
	})),
);
const CharactersPage = lazy(() => import("./components/CharactersPage"));
const EncountersPage = lazy(() => import("./components/EncountersPage"));
const NpcsPage = lazy(() => import("./components/NpcsPage"));
const FriendsPage = lazy(() => import("./components/FriendsPage"));

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
	const activeNav =
		NAV_ITEMS.find((item) => location.pathname.startsWith(item.path))?.path ??
		false;
	const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);

	return (
		<Box
			sx={{
				minHeight: "100vh",
				width: "100%",
				display: "flex",
				flexDirection: "column",
			}}
		>
			<AppBar position="sticky" color="default" elevation={0}>
				<Container maxWidth="lg" disableGutters>
					<Toolbar
						disableGutters
						sx={{
							px: { xs: 2, sm: 3 },
							gap: { xs: 1, md: 3 },
							minHeight: { xs: 56, sm: 64 },
						}}
					>
						<Typography
							variant="h6"
							component="h1"
							noWrap
							sx={{ fontSize: { xs: "1.05rem", sm: "1.25rem" } }}
						>
							Initiative Tracker
						</Typography>
						{/* Desktop: tabs with an underline indicator. Below md the
						    same items live in the menu at the far right. */}
						<Tabs
							value={activeNav}
							onChange={(_, path: string) => navigate(path)}
							sx={{
								display: { xs: "none", md: "flex" },
								flexGrow: 1,
								alignSelf: "stretch",
								"& .MuiTabs-flexContainer": { height: "100%" },
								"& .MuiTab-root": {
									minWidth: 0,
									px: 1.5,
									color: "text.secondary",
								},
								"& .MuiTabs-indicator": { height: 3, borderRadius: "3px 3px 0 0" },
							}}
						>
							{NAV_ITEMS.map((item) => (
								<Tab key={item.path} value={item.path} label={item.label} />
							))}
						</Tabs>
						<Box sx={{ flexGrow: 1, display: { xs: "block", md: "none" } }} />
						<Stack direction="row" spacing={1} alignItems="center">
							<ColorModeToggle />
							<AuthButtons />
							<IconButton
								aria-label="Open navigation"
								onClick={(e) => setMenuAnchor(e.currentTarget)}
								sx={{ display: { xs: "inline-flex", md: "none" } }}
							>
								<MenuIcon />
							</IconButton>
						</Stack>
						<Menu
							anchorEl={menuAnchor}
							open={menuAnchor !== null}
							onClose={() => setMenuAnchor(null)}
						>
							{NAV_ITEMS.map((item) => (
								<MenuItem
									key={item.path}
									selected={activeNav === item.path}
									onClick={() => {
										setMenuAnchor(null);
										navigate(item.path);
									}}
								>
									{item.label}
								</MenuItem>
							))}
						</Menu>
					</Toolbar>
				</Container>
			</AppBar>
			<Container
				maxWidth={maxWidth}
				sx={{
					py: { xs: 2, sm: 4 },
					px: { xs: 2, sm: 3 },
					textAlign: "left",
					flexGrow: 1,
				}}
			>
				{/* Thin bar while a page chunk loads (first visit only). */}
				<Suspense fallback={<LinearProgress sx={{ borderRadius: 1 }} />}>
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
				</Suspense>
			</Container>
			<Footer />
		</Box>
	);
}

export default App;
