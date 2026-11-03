import { FileText, Pause, Play, Square, Trash2 } from "lucide-react";
import type { TorrentInfo } from "../types/torrent";

type TorrentTableProps = {
  torrents: TorrentInfo[];
  busyID: string | null;
  selectedID: string | null;
  onPause: (torrent: TorrentInfo) => void;
  onRemove: (torrent: TorrentInfo) => void;
  onResume: (torrent: TorrentInfo) => void;
  onSelect: (id: string) => void;
  onStop: (torrent: TorrentInfo) => void;
};

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
  if (status === "Paused") return "paused";
  if (status === "Stopped") return "stopped";
  if (status === "Fetching metadata") return "metadata";
  if (status === "Waiting for peers") return "queued";
  return "downloading";
}

export function TorrentTable({ torrents, busyID, selectedID, onPause, onRemove, onResume, onSelect, onStop }: TorrentTableProps) {
  if (torrents.length === 0) {
    return <div className="tg-empty"><strong>Nothing here</strong>No torrents match this filter or search.</div>;
  }

  return (
    <div className="tg-list-inner">
      <div className="tg-list-grid tg-list-head"><span>Name</span><span>Progress</span><span>Down</span><span>Up</span><span>ETA</span><span /></div>
      {torrents.map((torrent) => {
        const busy = busyID === torrent.id;
        const status = statusClass(torrent.status);
        return (
          <div aria-selected={selectedID === torrent.id} className={`tg-list-grid tg-list-row ${selectedID === torrent.id ? "is-selected" : ""}`} key={torrent.id}>
            <button aria-label={`Select ${torrent.name}`} className="tg-name-button" onClick={() => onSelect(torrent.id)} type="button">
              <span className="tg-file-icon"><FileText size={18} /></span>
              <span className="tg-name-column"><span className="tg-name" title={torrent.name}>{torrent.name}</span><span className="tg-sub"><span className={`tg-status ${status}`}>{torrent.status}</span> · {torrent.progress >= 100 ? formatBytes(torrent.size) : `${formatBytes(torrent.downloaded)} of ${formatBytes(torrent.size)}`}</span></span>
            </button>
            <div className="tg-progress"><span className={`tg-meter ${status}`}><i style={{ width: `${torrent.progress}%` }} /></span><span className="tg-percent">{torrent.progress.toFixed(1)}%</span></div>
            <span className="tg-number">{formatRate(torrent.downloadRate)}</span>
            <span className="tg-number is-muted">{formatRate(torrent.uploadRate)}</span>
            <span className="tg-number is-muted">{formatETA(torrent.etaSeconds)}</span>
            <div className="tg-actions">
              {torrent.status === "Seeding" ? <button aria-label={`Stop seeding ${torrent.name}`} className="tg-icon-button" disabled={busy} onClick={() => onStop(torrent)} title="Stop seeding" type="button"><Square fill="currentColor" size={15} /></button>
                : torrent.paused || torrent.status === "Stopped" ? <button aria-label={`Resume ${torrent.name}`} className="tg-icon-button" disabled={busy} onClick={() => onResume(torrent)} title="Resume" type="button"><Play fill="currentColor" size={17} /></button>
                  : <button aria-label={`Pause ${torrent.name}`} className="tg-icon-button" disabled={busy} onClick={() => onPause(torrent)} title="Pause" type="button"><Pause size={17} /></button>}
              <button aria-label={`Remove ${torrent.name}`} className="tg-icon-button" disabled={busy} onClick={() => onRemove(torrent)} title="Remove from TorrentGo" type="button"><Trash2 size={17} /></button>
            </div>
          </div>
        );
      })}
    </div>
  );
}
