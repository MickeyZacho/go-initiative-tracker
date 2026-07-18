import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vite.dev/config/
export default defineConfig({
	// Bake the git short SHA into the bundle at build time. Supplied via the
	// GIT_SHA env var (set by frontend/Dockerfile from a build arg, or in the
	// shell for a local build); falls back to "dev".
	define: {
		__APP_VERSION__: JSON.stringify(process.env.GIT_SHA || "dev"),
	},
	plugins: [
		react({
			babel: {
				plugins: [["babel-plugin-react-compiler"]],
			},
		}),
	],
	server: {
		proxy: {
			"/api": {
				target: "http://backend:8080",
				changeOrigin: true,
				rewrite: (path) => path.replace(/^\/api/, ""),
			},
		},
		watch: {
			usePolling: true,
		},
	},
});
