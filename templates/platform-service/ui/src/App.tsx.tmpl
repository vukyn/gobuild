import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { Flex, Spinner } from "@chakra-ui/react";

// Route-level code splitting: every page is its own lazily-loaded chunk so the
// initial bundle doesn't ship the whole app (each page loads only when navigated
// to). React.lazy expects a default export; pages defined as named exports map
// it via `.then((m) => ({ default: m.PageName }))`. Add real pages under
// ./pages/ and register them here as the app grows — keep this lazy + Suspense
// shape so the manualChunks split in vite.config.ts stays effective.
const Home = lazy(() =>
	import("./pages/Home").then((m) => ({ default: m.Home }))
);

/** Full-screen fallback shown while a route chunk loads. */
const RouteFallback = () => (
	<Flex minH="100dvh" align="center" justify="center">
		<Spinner size="lg" />
	</Flex>
);

function App() {
	return (
		<BrowserRouter>
			<Suspense fallback={<RouteFallback />}>
				<Routes>
					<Route path="/" element={<Home />} />
					<Route path="*" element={<Navigate to="/" replace />} />
				</Routes>
			</Suspense>
		</BrowserRouter>
	);
}

export default App;
