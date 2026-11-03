import { useState } from "react";
import {
  Download,
  FolderOpen,
  Link,
  Pause,
  Play,
  Plus,
  Radio,
  ShieldCheck,
  Upload,
} from "lucide-react";
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

function messageFrom(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function App() {
  const { error, loading, refresh, setError, setSettings, settings, snapshot } = useTorrents();
  const [magnet, setMagnet] = useState("");
  const [busyID, setBusyID] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const run = async (action: () => Promise<unknown>, busy?: string) => {
    setError(null);
    if (busy) setBusyID(busy);
    else setSubmitting(true);
    try {
      await action();
      await refresh();
    } catch (cause) {
      setError(messageFrom(cause));
    } finally {
      if (busy) setBusyID(null);
      else setSubmitting(false);
    }
  };

  const addMagnet = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!magnet.trim()) {
      setError("Paste a magnet link before adding it.");
      return;
    }
    await run(async () => {
      await torrentService.addMagnet(magnet.trim());
      setMagnet("");
    });
  };

  const chooseDirectory = async () => {
    await run(async () => {
      const nextSettings = await torrentService.chooseDownloadDirectory();
      setSettings(nextSettings);
    });
  };

  const removeTorrent = async (torrent: TorrentInfo) => {
    const confirmed = window.confirm(`Remove “${torrent.name}” from TorrentGo? Downloaded files will be kept.`);
    if (!confirmed) return;
    await run(() => torrentService.remove(torrent.id), torrent.id);
  };

  return (
    <main className="min-h-screen bg-slate-950 text-slate-100">
      <div className="mx-auto flex min-h-screen max-w-400 flex-col px-5 py-5 lg:px-8">
        <header className="flex flex-col gap-5 border-b border-slate-800 pb-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex items-center gap-3">
            <div className="grid size-10 place-items-center rounded-xl bg-sky-400 text-slate-950 shadow-lg shadow-sky-400/10"><Download size={21} strokeWidth={2.5} /></div>
            <div>
              <h1 className="text-lg font-semibold tracking-tight">TorrentGo</h1>
              <p className="text-xs text-slate-500">Local-first BitTorrent client</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <span className={`size-2 rounded-full ${snapshot.networkReady ? "bg-emerald-400" : "bg-slate-600"}`} />
            {snapshot.networkReady ? "Engine online" : "Connecting to engine"}
          </div>
        </header>

        <section className="flex flex-col gap-3 py-5 xl:flex-row xl:items-center">
          <button className="inline-flex items-center justify-center gap-2 rounded-lg bg-sky-400 px-3.5 py-2.5 text-sm font-semibold text-slate-950 transition hover:bg-sky-300 disabled:bg-slate-700 disabled:text-slate-300" disabled={submitting} onClick={() => void run(() => torrentService.addTorrentFile())} type="button">
            <Plus size={17} /> Add .torrent
          </button>
          <form className="flex min-w-0 flex-1 gap-2" onSubmit={(event) => void addMagnet(event)}>
            <label className="sr-only" htmlFor="magnet">Magnet link</label>
            <div className="relative min-w-0 flex-1">
              <Link className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" size={16} />
              <input className="w-full rounded-lg border border-slate-800 bg-slate-900 py-2.5 pl-10 pr-3 text-sm text-slate-100 outline-none placeholder:text-slate-600 focus:border-sky-400/70" id="magnet" onChange={(event) => setMagnet(event.target.value)} placeholder="Paste a magnet link" value={magnet} />
            </div>
            <button className="rounded-lg border border-slate-700 px-3.5 text-sm font-medium text-slate-200 transition hover:border-sky-400/70 hover:text-sky-300 disabled:text-slate-600" disabled={submitting} type="submit">Add magnet</button>
          </form>
          <div className="flex shrink-0 gap-2">
            <button aria-label="Resume all torrents" className="icon-button border border-slate-800" disabled={submitting || snapshot.torrents.length === 0} onClick={() => void run(() => torrentService.resumeAll())} title="Resume all" type="button"><Play size={16} /></button>
            <button aria-label="Pause all torrents" className="icon-button border border-slate-800" disabled={submitting || snapshot.torrents.length === 0} onClick={() => void run(() => torrentService.pauseAll())} title="Pause all" type="button"><Pause size={16} /></button>
          </div>
        </section>

        <section className="mb-5 flex flex-col gap-3 rounded-xl border border-slate-800 bg-slate-900/45 p-3.5 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex min-w-0 items-center gap-3">
            <FolderOpen className="shrink-0 text-sky-300" size={18} />
            <div className="min-w-0">
              <p className="text-xs font-medium uppercase tracking-wide text-slate-500">New downloads</p>
              <p className="truncate text-sm text-slate-200" title={settings.downloadDirectory}>{settings.downloadDirectory || "Loading folder…"}</p>
            </div>
          </div>
          <button className="shrink-0 rounded-lg border border-slate-700 px-3 py-2 text-sm font-medium text-slate-300 transition hover:border-sky-400/70 hover:text-sky-300 disabled:text-slate-600" disabled={submitting} onClick={() => void chooseDirectory()} type="button">Choose folder</button>
        </section>

        {error && <div className="mb-5 rounded-lg border border-rose-400/25 bg-rose-400/10 px-3.5 py-3 text-sm text-rose-200" role="alert">{error}</div>}

        <section className="flex-1">
          <div className="mb-3 flex items-center justify-between">
            <div>
              <h2 className="font-semibold">All torrents</h2>
              <p className="mt-0.5 text-xs text-slate-500">{loading ? "Loading local state…" : `${snapshot.torrents.length} tracked locally`}</p>
            </div>
            <span className="rounded-full bg-slate-900 px-2.5 py-1 text-xs text-slate-400">{snapshot.active} active</span>
          </div>
          <TorrentTable busyID={busyID} onPause={(torrent) => void run(() => torrentService.pause(torrent.id), torrent.id)} onRemove={(torrent) => void removeTorrent(torrent)} onResume={(torrent) => void run(() => torrentService.resume(torrent.id), torrent.id)} onStop={(torrent) => void run(() => torrentService.stop(torrent.id), torrent.id)} torrents={snapshot.torrents} />
        </section>

        <footer className="mt-5 flex flex-col gap-3 border-t border-slate-800 pt-4 text-xs text-slate-500 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-4">
            <span className="inline-flex items-center gap-1.5"><Download size={14} className="text-sky-300" /> {formatRate(snapshot.downloadRate)}</span>
            <span className="inline-flex items-center gap-1.5"><Upload size={14} className="text-emerald-300" /> {formatRate(snapshot.uploadRate)}</span>
            <span className="inline-flex items-center gap-1.5"><Radio size={14} /> {snapshot.active} active</span>
          </div>
          <span className="inline-flex items-center gap-1.5"><ShieldCheck size={14} className="text-slate-400" /> No telemetry · No cloud downloads</span>
        </footer>
      </div>
    </main>
  );
}

export default App;
