import { useDeferredValue } from "react";
import { AppShell } from "../components/layout/AppShell";
import { ConnectionsPage } from "../pages/ConnectionsPage";
import { EventsPage } from "../pages/EventsPage";
import { SettingsPage } from "../pages/SettingsPage";
import { useSettings } from "./useSettings";
import { useEvents } from "./useEvents";
import { useUrlNavigation } from "./useUrlFilters";

export function App() {
  const { view, setView, filters, setFilters } = useUrlNavigation();
  const { settings } = useSettings();
  const deferredFilters = useDeferredValue(filters);
  const events = useEvents(settings.apiBaseUrl, deferredFilters);
  const streamLive = events.state.kind === "ready";

  return (
    <AppShell view={view} onNavigate={setView} streamLive={streamLive}>
      {view === "events" && (
        <EventsPage filters={filters} onFiltersChange={setFilters} events={events} />
      )}
      {view === "connections" && <ConnectionsPage />}
      {view === "settings" && <SettingsPage />}
    </AppShell>
  );
}
