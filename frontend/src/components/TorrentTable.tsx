import { Pause, Play, Trash2, Users } from "lucide-react";
import { ProgressBar } from "./ProgressBar";
import type { TorrentInfo } from "../types/torrent";

type TorrentTableProps = {
  torrents: TorrentInfo[];
  busyID: string | null;
  onPause: (torrent: TorrentInfo) => void;
  onRemove: (torrent: TorrentInfo) => void;
  onResume: (torrent: TorrentInfo) => void;
};

function formatBytes(value: number): string {
  if (!value) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(
    Math.floor(Math.log(value) / Math.log(1024)),
    units.length - 1,
  );
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function formatRate(value: number): string {
  return value > 0 ? `${formatBytes(value)}/s` : "—";
}

function formatETA(seconds: number): string {
  if (seconds <= 0 || !Number.isFinite(seconds)) return "—";
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remainingSeconds = Math.floor(seconds % 60);
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${remainingSeconds}s`;
  return `${remainingSeconds}s`;
}

function statusClass(status: string): string {
  if (status === "Seeding") return "bg-emerald-400/10 text-emerald-300";
  if (status === "Paused") return "bg-amber-400/10 text-amber-300";
  if (status === "Fetching metadata") return "bg-violet-400/10 text-violet-300";
  return "bg-sky-400/10 text-sky-300";
}

export function TorrentTable({
  torrents,
  busyID,
  onPause,
  onRemove,
  onResume,
}: TorrentTableProps) {
  if (torrents.length === 0) {
    return (
      <div className="grid min-h-80 place-items-center rounded-xl border border-dashed border-slate-800 bg-slate-900/30 px-6 text-center">
        <div>
          <p className="text-base font-medium text-slate-200">
            No torrents yet
          </p>
          <p className="mt-2 max-w-sm text-sm leading-6 text-slate-500">
            Add a legal .torrent file or paste a magnet link to begin a local
            download.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-slate-800 bg-slate-900/40">
      <table className="w-full min-w-[900px] border-collapse text-left text-sm">
        <thead className="border-b border-slate-800 bg-slate-900/80 text-xs uppercase tracking-wide text-slate-500">
          <tr>
            <th className="px-4 py-3 font-medium">Name</th>
            <th className="px-4 py-3 font-medium">Size</th>
            <th className="min-w-45 px-4 py-3 font-medium">Progress</th>
            <th className="px-4 py-3 font-medium">Status</th>
            <th className="px-4 py-3 font-medium">Down</th>
            <th className="px-4 py-3 font-medium">Up</th>
            <th className="px-4 py-3 font-medium">ETA</th>
            <th className="px-4 py-3 font-medium">
              <span className="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800/80">
          {torrents.map((torrent) => {
            const busy = busyID === torrent.id;
            return (
              <tr
                className="transition-colors hover:bg-slate-800/30"
                key={torrent.id}
              >
                <td className="max-w-72 px-4 py-4">
                  <p
                    className="truncate font-medium text-slate-100"
                    title={torrent.name}
                  >
                    {torrent.name}
                  </p>
                  <p
                    className="mt-1 truncate text-xs text-slate-500"
                    title={torrent.downloadDirectory}
                  >
                    {torrent.downloadDirectory}
                  </p>
                </td>
                <td className="whitespace-nowrap px-4 py-4 text-slate-300">
                  {formatBytes(torrent.size)}
                </td>
                <td className="px-4 py-4">
                  <div className="flex items-center gap-2.5">
                    <ProgressBar value={torrent.progress} />
                    <span className="w-11 text-right text-xs tabular-nums text-slate-400">
                      {torrent.progress.toFixed(1)}%
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-slate-500">
                    {formatBytes(torrent.downloaded)} of{" "}
                    {formatBytes(torrent.size)}
                  </p>
                </td>
                <td className="px-4 py-4">
                  <span
                    className={`inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-xs font-medium ${statusClass(torrent.status)}`}
                  >
                    <Users size={12} /> {torrent.status}
                  </span>
                  {torrent.peers > 0 && (
                    <p className="mt-1 text-xs text-slate-500">
                      {torrent.peers} peers · {torrent.seeds} seeds
                    </p>
                  )}
                </td>
                <td className="whitespace-nowrap px-4 py-4 tabular-nums text-slate-300">
                  {formatRate(torrent.downloadRate)}
                </td>
                <td className="whitespace-nowrap px-4 py-4 tabular-nums text-slate-300">
                  {formatRate(torrent.uploadRate)}
                </td>
                <td className="whitespace-nowrap px-4 py-4 tabular-nums text-slate-300">
                  {formatETA(torrent.etaSeconds)}
                </td>
                <td className="px-4 py-4">
                  <div className="flex items-center justify-end gap-1">
                    {torrent.paused ? (
                      <button
                        aria-label={`Resume ${torrent.name}`}
                        className="icon-button"
                        disabled={busy}
                        onClick={() => onResume(torrent)}
                        type="button"
                      >
                        <Play size={16} />
                      </button>
                    ) : (
                      <button
                        aria-label={`Pause ${torrent.name}`}
                        className="icon-button"
                        disabled={busy}
                        onClick={() => onPause(torrent)}
                        type="button"
                      >
                        <Pause size={16} />
                      </button>
                    )}
                    <button
                      aria-label={`Remove ${torrent.name}`}
                      className="icon-button text-slate-400 hover:text-rose-300"
                      disabled={busy}
                      onClick={() => onRemove(torrent)}
                      type="button"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
