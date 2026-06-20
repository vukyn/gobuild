import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// https://vite.dev/config/
export default defineConfig({
	base: "/",
	plugins: [react()],
	server: {
		host: true,
	},
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},
	build: {
		minify: "esbuild",
		sourcemap: false,
		reportCompressedSize: true,
		chunkSizeWarningLimit: 1000,
		rollupOptions: {
			output: {
				// Split the large/shared vendor libs into their own long-cached chunks
				// (named by package dir) so they're not duplicated across route chunks
				// and don't bloat the entry. Route-level code is split via React.lazy.
				manualChunks(id) {
					if (!id.includes("node_modules/")) {
						return;
					}
					// Group react-router + react-router-dom into one chunk — splitting them
					// separately leaves react-router-dom an empty (re-export-only) chunk.
					if (id.includes("node_modules/react-router")) {
						return "react-router";
					}
					const libraries = ["@chakra-ui", "react-icons", "axios"];
					if (libraries.some((lib) => id.includes(`node_modules/${lib}`))) {
						return id.toString().split("node_modules/")[1].split("/")[0].toString();
					}
				},
			},
		},
	},
});
