import type { TorrentInfo } from "../types/torrent";

type TorrentDetailsProps = { torrent: TorrentInfo | undefined };

function formatBytes(value: number): string {
  if (!value) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}
function formatRate(value: number): string { return value > 0 ? `${formatBytes(value)}/s` : "—"; }
function formatETA(seconds: number): string {
  if (seconds <= 0) return "—";
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
}
function DetailItem({ label, value }: { label: string; value: string }) { return <div><dt>{label}</dt><dd title={value}>{value}</dd></div>; }

export function TorrentDetails({ torrent }: TorrentDetailsProps) {
  if (!torrent) return <section className="tg-card tg-details"><div className="tg-details-empty">Select a torrent to inspect its live transfer information.</div></section>;
  const meterClass = torrent.status === "Seeding" ? "seed" : torrent.status === "Paused" ? "paused" : torrent.status === "Stopped" ? "stopped" : "";
  return (
    <section aria-label="Torrent details" className="tg-card tg-details">
      <div className="tg-details-head"><div className="tg-details-title" title={torrent.name}>{torrent.name}</div><button aria-selected className="tg-detail-tab" role="tab" type="button">Overview</button></div>
      <div className="tg-detail-body">
        <div className="tg-detail-progress-head"><span>Progress</span><span>{torrent.progress.toFixed(1)}% complete</span></div>
        <div className={`tg-meter tg-piece-meter ${meterClass}`}><i style={{ width: `${torrent.progress}%` }} /></div>
        <dl className="tg-facts">
          <DetailItem label="Save to" value={torrent.downloadDirectory} />
          <DetailItem label="Status" value={torrent.status} />
          <DetailItem label="Size" value={formatBytes(torrent.size)} />
          <DetailItem label="Downloaded" value={formatBytes(torrent.downloaded)} />
          <DetailItem label="Uploaded" value={formatBytes(torrent.uploaded)} />
          <DetailItem label="Remaining" value={formatBytes(torrent.remaining)} />
          <DetailItem label="Download speed" value={formatRate(torrent.downloadRate)} />
          <DetailItem label="Upload speed" value={formatRate(torrent.uploadRate)} />
          <DetailItem label="ETA" value={formatETA(torrent.etaSeconds)} />
          <DetailItem label="Seeds · Peers" value={`${torrent.seeds} · ${torrent.peers}`} />
        </dl>
      </div>
    </section>
  );
}
