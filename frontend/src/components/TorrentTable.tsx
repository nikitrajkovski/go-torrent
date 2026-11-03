import { useEffect, useRef, useState } from "react";
import { CheckCircle2, FileText, FolderOpen, MoreHorizontal, Pause, Play, Square, Trash2 } from "lucide-react";
import type { TorrentInfo } from "../types/torrent";

type TorrentTableProps = {
  torrents: TorrentInfo[];
  busyID: string | null;
  selectedID: string | null;
  onOpenFolder: (torrent: TorrentInfo) => void;
  onPause: (torrent: TorrentInfo) => void;
  onRemove: (torrent: TorrentInfo) => void;
  onResume: (torrent: TorrentInfo) => void;
  onSelect: (id: string) => void;
  onStop: (torrent: TorrentInfo) => void;
};

type ContextMenu = { torrent: TorrentInfo; x: number; y: number } | null;

function formatBytes(value: number): string {
  if (!value) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function formatRate(value: number): string { return value > 0 ? `${formatBytes(value)}/s` : "—"; }
function formatETA(seconds: number): string {
  if (seconds <= 0 || !Number.isFinite(seconds)) return "—";
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m ${Math.floor(seconds % 60)}s`;
}

function statusClass(status: string): string {
  if (status === "Seeding") return "seeding";
  if (status === "Completed") return "completed";
  if (status === "Paused") return "paused";
  if (status === "Stopped") return "stopped";
  if (status === "Fetching metadata") return "metadata";
  if (status === "Awaiting file selection") return "queued";
  if (status === "Waiting for peers") return "queued";
  return "downloading";
}

function transferAction(torrent: TorrentInfo, callbacks: Pick<TorrentTableProps, "onPause" | "onResume" | "onStop">): { label: string; run: () => void } {
  if (torrent.status === "Seeding") return { label: "Stop seeding", run: () => callbacks.onStop(torrent) };
  if (torrent.paused || torrent.status === "Stopped" || torrent.status === "Completed") return { label: "Resume", run: () => callbacks.onResume(torrent) };
  return { label: "Pause", run: () => callbacks.onPause(torrent) };
}

export function TorrentTable({ torrents, busyID, selectedID, onOpenFolder, onPause, onRemove, onResume, onSelect, onStop }: TorrentTableProps) {
  const [contextMenu, setContextMenu] = useState<ContextMenu>(null);
  const contextMenuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const close = () => setContextMenu(null);
    const closeOutsideMenu = (event: PointerEvent) => {
      if (!contextMenuRef.current?.contains(event.target as Node)) close();
    };
    const closeOnEscape = (event: KeyboardEvent) => { if (event.key === "Escape") close(); };
    window.addEventListener("pointerdown", closeOutsideMenu);
    window.addEventListener("blur", close);
    window.addEventListener("keydown", closeOnEscape);
    return () => {
      window.removeEventListener("pointerdown", closeOutsideMenu);
      window.removeEventListener("blur", close);
      window.removeEventListener("keydown", closeOnEscape);
    };
  }, []);

  if (torrents.length === 0) {
    return <div className="tg-empty"><strong>Nothing here</strong>No torrents match this filter or search.</div>;
  }

  const showMenu = (torrent: TorrentInfo, x: number, y: number) => {
    onSelect(torrent.id);
    setContextMenu({ torrent, x: Math.max(8, Math.min(x, window.innerWidth - 208)), y: Math.max(8, Math.min(y, window.innerHeight - 180)) });
  };
  const runMenuAction = (action: () => void) => { setContextMenu(null); action(); };

  return (
    <div className="tg-list-inner">
      <div className="tg-list-grid tg-list-head"><span>Name</span><span className="tg-progress-heading">Progress</span><span>Down</span><span>Up</span><span>ETA</span><span /></div>
      {torrents.map((torrent) => {
        const busy = busyID === torrent.id;
        const status = statusClass(torrent.status);
        const completed = torrent.progress >= 100;
        return (
          <div aria-selected={selectedID === torrent.id} className={`tg-list-grid tg-list-row ${selectedID === torrent.id ? "is-selected" : ""}`} key={torrent.id} onContextMenu={(event) => { event.preventDefault(); showMenu(torrent, event.clientX, event.clientY); }}>
            <button aria-label={`Select ${torrent.name}`} className="tg-name-button" onClick={() => onSelect(torrent.id)} onDoubleClick={() => { if (completed) onOpenFolder(torrent); }} title={completed ? "Double-click to show the download folder" : undefined} type="button">
              <span className={`tg-file-icon ${completed ? "is-complete" : ""}`}>{completed ? <CheckCircle2 size={18} /> : <FileText size={18} />}</span>
              <span className="tg-name-column"><span className="tg-name" title={torrent.name}>{torrent.name}</span><span className="tg-sub"><span className={`tg-status ${status}`}>{torrent.status}</span> · {torrent.status === "Fetching metadata" && torrent.size === 0 ? "Restoring local torrent data…" : torrent.progress >= 100 ? formatBytes(torrent.size) : `${formatBytes(torrent.downloaded)} of ${formatBytes(torrent.size)}`}</span></span>
            </button>
            <div className="tg-progress"><span className={`tg-meter ${status}`}><i style={{ width: `${torrent.progress}%` }} /></span><span className="tg-percent">{torrent.progress.toFixed(1)}%</span></div>
            <span className="tg-number">{formatRate(torrent.downloadRate)}</span>
            <span className="tg-number is-muted">{formatRate(torrent.uploadRate)}</span>
            <span className="tg-number is-muted">{formatETA(torrent.etaSeconds)}</span>
            <div className="tg-actions">
              {torrent.status === "Seeding" ? <button aria-label={`Stop seeding ${torrent.name}`} className="tg-icon-button" disabled={busy} onClick={() => onStop(torrent)} title="Stop seeding" type="button"><Square fill="currentColor" size={15} /></button>
                : torrent.paused || torrent.status === "Stopped" || torrent.status === "Completed" ? <button aria-label={`Resume ${torrent.name}`} className="tg-icon-button" disabled={busy} onClick={() => onResume(torrent)} title="Resume seeding" type="button"><Play fill="currentColor" size={17} /></button>
                  : <button aria-label={`Pause ${torrent.name}`} className="tg-icon-button" disabled={busy} onClick={() => onPause(torrent)} title="Pause" type="button"><Pause size={17} /></button>}
              <button aria-label={`More actions for ${torrent.name}`} aria-expanded={contextMenu?.torrent.id === torrent.id} className="tg-icon-button" disabled={busy} onClick={(event) => showMenu(torrent, event.clientX, event.clientY)} title="More actions" type="button"><MoreHorizontal size={18} /></button>
            </div>
          </div>
        );
      })}
      {contextMenu && <div className="tg-context-menu" ref={contextMenuRef} role="menu" style={{ left: contextMenu.x, top: contextMenu.y }}>
        <button onClick={() => runMenuAction(() => onOpenFolder(contextMenu.torrent))} role="menuitem" type="button"><FolderOpen size={16} /> Show in folder</button>
        <button onClick={() => runMenuAction(transferAction(contextMenu.torrent, { onPause, onResume, onStop }).run)} role="menuitem" type="button">{contextMenu.torrent.status === "Seeding" ? <Square size={15} /> : contextMenu.torrent.paused || contextMenu.torrent.status === "Stopped" || contextMenu.torrent.status === "Completed" ? <Play size={16} /> : <Pause size={16} />}{transferAction(contextMenu.torrent, { onPause, onResume, onStop }).label}</button>
        <button className="is-danger" onClick={() => runMenuAction(() => onRemove(contextMenu.torrent))} role="menuitem" type="button"><Trash2 size={16} /> Remove from Torrade</button>
      </div>}
    </div>
  );
}
