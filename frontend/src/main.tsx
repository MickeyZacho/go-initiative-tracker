import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { CssBaseline } from "@mui/material";
import "./index.css";
import App from "./App.tsx";
import { ThemeTokensProvider } from "./components/ThemeTokensProvider";

createRoot(document.getElementById("root")!).render(
	<StrictMode>
		<ThemeTokensProvider>
			<CssBaseline />
			<BrowserRouter>
				<App />
			</BrowserRouter>
		</ThemeTokensProvider>
	</StrictMode>,
);
