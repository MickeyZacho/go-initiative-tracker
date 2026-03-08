import { CharacterList } from "./components/CharacterList";
import AuthButtons from "./components/AuthButtons";

function App() {
	return (
		<div style={{ padding: "2rem" }}>
			<AuthButtons />
			<h1>Initiative Tracker</h1>
			<CharacterList />
		</div>
	);
}

export default App;
