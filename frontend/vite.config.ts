import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
	plugins: [
		react({
			babel: {
				plugins: [["babel-plugin-react-compiler"]],
			},
		}),
	],
	server: {
		proxy: {
			"/api": "http://localhost:8080",
			"/save-character": "http://localhost:8080",
			"/add-character-to-encounter": "http://localhost:8080",
			"/remove-character-from-encounter": "http://localhost:8080",
			"/login/discord": "http://localhost:8080",
			"/logout": "http://localhost:8080",
			"/auth/discord/callback": "http://localhost:8080",
		},
		watch: {
			usePolling: true,
		},
	},
});
