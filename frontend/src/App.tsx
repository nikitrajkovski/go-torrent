import { useMemo, useState } from "react";
import { Download, Link, Pause, Play, Search, ShieldCheck, Upload, X } from "lucide-react";
import { SettingsPanel } from "./components/SettingsPanel";
import { Sidebar, type TorrentView } from "./components/Sidebar";
import { TorrentDetails } from "./components/TorrentDetails";
import { TorrentTable } from "./components/TorrentTable";
import { useTorrents } from "./hooks/useTorrents";
import { torrentService } from "./services/torrent-service";
import type { TorrentInfo } from "./types/torrent";

function formatRate(value: number): string {
  if (value <= 0) return "0 B/s";
  const units = ["B", "KB", "MB", "GB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}/s`;
}
function messageFrom(error: unknown): string { return error instanceof Error ? error.message : String(error); }
function matchesView(torrent: TorrentInfo, view: TorrentView): boolean {
  switch (view) {
    case "all": return true;
    case "downloading": return ["Fetching metadata", "Waiting for peers", "Downloading"].includes(torrent.status);
    case "seeding": return torrent.status === "Seeding";
    case "completed": return torrent.progress >= 100;
    case "paused": return torrent.status === "Paused";
    case "settings": return false;
  }
}

function App() {
  const { error, loading, refresh, setError, setSettings, settings, snapshot } = useTorrents();
  const [activeView, setActiveView] = useState<TorrentView>("all");
  const [magnet, setMagnet] = useState("");
  const [query, setQuery] = useState("");
  const [busyID, setBusyID] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const counts = useMemo(() => ({ all: snapshot.torrents.length, downloading: snapshot.torrents.filter((torrent) => matchesView(torrent, "downloading")).length, seeding: snapshot.torrents.filter((torrent) => matchesView(torrent, "seeding")).length, completed: snapshot.torrents.filter((torrent) => matchesView(torrent, "completed")).length, paused: snapshot.torrents.filter((torrent) => matchesView(torrent, "paused")).length }), [snapshot.torrents]);
  const displayedTorrents = useMemo(() => {
    const search = query.trim().toLocaleLowerCase();
    return snapshot.torrents.filter((torrent) => matchesView(torrent, activeView) && (!search || torrent.name.toLocaleLowerCase().includes(search)));
  }, [activeView, query, snapshot.torrents]);
  const selectedTorrent = snapshot.torrents.find((torrent) => torrent.id === selectedID);

  const run = async (action: () => Promise<unknown>, busy?: string) => {
    setError(null);
    if (busy) setBusyID(busy); else setSubmitting(true);
    try { await action(); await refresh(); } catch (cause) { setError(messageFrom(cause)); } finally { if (busy) setBusyID(null); else setSubmitting(false); }
  };
  const addMagnet = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!magnet.trim()) { setError("Paste a magnet link before adding it."); return; }
    await run(async () => { const torrent = await torrentService.addMagnet(magnet.trim()); setMagnet(""); setSelectedID(torrent.id); setActiveView("all"); setAddDialogOpen(false); });
  };
  const addTorrentFile = async () => run(async () => { await torrentService.addTorrentFile(); setAddDialogOpen(false); });
  const chooseDirectory = async () => run(async () => setSettings(await torrentService.chooseDownloadDirectory()));
  const changeTheme = async (theme: "dark" | "light") => { if (theme !== settings.theme) await run(async () => setSettings(await torrentService.updateSettings({ ...settings, theme }))); };
  const removeTorrent = async (torrent: TorrentInfo) => {
    if (!window.confirm(`Remove “${torrent.name}” from TorrentGo? Downloaded files will be kept.`)) return;
    await run(async () => { await torrentService.remove(torrent.id); if (selectedID === torrent.id) setSelectedID(null); }, torrent.id);
  };

  return (
    <main className="tg-app" data-file-drop-target data-theme={settings.theme} id="torrent-drop-zone">
      <Sidebar activeView={activeView} counts={counts} onAdd={() => setAddDialogOpen(true)} onSelect={setActiveView} />
      <div className="tg-main">
        <header className="tg-topbar">
          <label className="tg-search"><span className="sr-only">Search torrents</span><Search size={18} /><input onChange={(event) => setQuery(event.target.value)} placeholder="Search your torrents" type="search" value={query} /></label>
          <div aria-label="Global transfer speed" className="tg-speeds" role="status"><span className="tg-speed-pill"><Download color="var(--tg-accent)" size={16} />{formatRate(snapshot.downloadRate)}</span><span className="tg-speed-pill"><Upload color="var(--tg-green)" size={16} />{formatRate(snapshot.uploadRate)}</span><button aria-label="Resume all torrents" className="tg-icon-button" disabled={submitting || snapshot.torrents.length === 0} onClick={() => void run(() => torrentService.resumeAll())} title="Resume all" type="button"><Play size={17} /></button><button aria-label="Pause all torrents" className="tg-icon-button" disabled={submitting || snapshot.torrents.length === 0} onClick={() => void run(() => torrentService.pauseAll())} title="Pause all" type="button"><Pause size={17} /></button></div>
        </header>
        {activeView === "settings" ? <SettingsPanel busy={submitting} onChooseDirectory={() => void chooseDirectory()} onThemeChange={(theme) => void changeTheme(theme)} settings={settings} /> : <div className="tg-content"><div className="tg-title-row"><h1>{activeView === "all" ? "All torrents" : activeView[0].toUpperCase() + activeView.slice(1)}</h1><span>{loading ? "Loading local state…" : `${displayedTorrents.length} torrents · ${snapshot.active} active`}</span></div><section aria-label="Torrents" className="tg-card tg-list"><TorrentTable busyID={busyID} onPause={(torrent) => void run(() => torrentService.pause(torrent.id), torrent.id)} onRemove={(torrent) => void removeTorrent(torrent)} onResume={(torrent) => void run(() => torrentService.resume(torrent.id), torrent.id)} onSelect={setSelectedID} onStop={(torrent) => void run(() => torrentService.stop(torrent.id), torrent.id)} selectedID={selectedID} torrents={displayedTorrents} /></section><TorrentDetails torrent={selectedTorrent} /></div>}
        {error && <div className="tg-alert" role="alert">{error}</div>}
        <footer className="tg-footer"><span><ShieldCheck size={13} /> No telemetry · No cloud downloads</span><span>{snapshot.networkReady ? "Engine online" : "Connecting to engine"}</span></footer>
      </div>
      {addDialogOpen && <div aria-modal className="tg-modal-overlay" onMouseDown={(event) => { if (event.target === event.currentTarget) setAddDialogOpen(false); }} role="dialog"><section className="tg-dialog"><div className="tg-dialog-head"><h2>Add torrent</h2><button aria-label="Close" className="tg-icon-button" onClick={() => setAddDialogOpen(false)} type="button"><X size={18} /></button></div><div className="tg-drop"><Download color="var(--tg-accent)" size={28} /><strong>Drop a .torrent file here</strong><button className="tg-secondary" disabled={submitting} onClick={() => void addTorrentFile()} type="button">Browse files</button></div><div className="tg-dialog-or">or</div><form onSubmit={(event) => void addMagnet(event)}><label className="tg-field-label" htmlFor="magnet">Magnet link</label><div className="tg-dialog-form"><input className="tg-input" id="magnet" onChange={(event) => setMagnet(event.target.value)} placeholder="magnet:?xt=urn:btih:…" value={magnet} /><button className="tg-primary" disabled={submitting} type="submit"><Link size={16} /> Add</button></div></form></section></div>}
    </main>
  );
}

export default App;
