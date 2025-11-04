import { CharacterList } from "./components/CharacterList";
import type { Character } from "./components/CharacterList";

const dummyCharacters: Character[] = [
	{
		ID: 1,
		Name: "Aragorn",
		ArmorClass: 16,
		MaxHP: 45,
		CurrentHP: 38,
		Initiative: 12,
		IsActive: true,
		OwnerID: "user1",
	},
	{
		ID: 2,
		Name: "Legolas",
		ArmorClass: 15,
		MaxHP: 40,
		CurrentHP: 40,
		Initiative: 18,
		IsActive: false,
		OwnerID: "user2",
	},
	{
		ID: 3,
		Name: "Gimli",
		ArmorClass: 17,
		MaxHP: 50,
		CurrentHP: 50,
		Initiative: 10,
		IsActive: false,
		OwnerID: "user3",
	},
];

const enemies: Character[] = [
	{
		ID: 101,
		Name: "Goblin",
		ArmorClass: 13,
		MaxHP: 7,
		CurrentHP: 7,
		Initiative: 14,
		IsActive: false,
		OwnerID: "enemy",
	},
	{
		ID: 102,
		Name: "Orc",
		ArmorClass: 15,
		MaxHP: 15,
		CurrentHP: 15,
		Initiative: 11,
		IsActive: false,
		OwnerID: "enemy",
	},
	{
		ID: 103,
		Name: "Dragon",
		ArmorClass: 19,
		MaxHP: 200,
		CurrentHP: 200,
		Initiative: 20,
		IsActive: false,
		OwnerID: "enemy",
	},
];

function App() {
	return (
		<div style={{ padding: "2rem" }}>
			<h1>Initiative Tracker</h1>
			<CharacterList
				initialCharacters={dummyCharacters}
				enemies={enemies}
			/>
		</div>
	);
}

export default App;
