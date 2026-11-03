import { useEffect, useState } from "react";
import { FileText, Users } from "lucide-react";
import { torrentService } from "../services/torrent-service";
import type { TorrentDetails as TorrentDetailsModel, TorrentFile, TorrentInfo } from "../types/torrent";

type DetailTab = "overview" | "files" | "peers" | "trackers";
type TorrentDetailsProps = { torrent: TorrentInfo | undefined; onFileSelected: (index: number, selected: boolean) => Promise<void> };

function formatBytes(value: number): string {
  if (!value) return "—";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}
function formatRate(value: number): string { return value > 0 ? `${formatBytes(value)}/s` : "—"; }
function formatETA(seconds: number): string { if (seconds <= 0) return "—"; const hours = Math.floor(seconds / 3600); const minutes = Math.floor((seconds % 3600) / 60); return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`; }
function DetailItem({ label, value }: { label: string; value: string }) { return <div><dt>{label}</dt><dd title={value}>{value}</dd></div>; }

function FileRow({ file, onChange }: { file: TorrentFile; onChange: (selected: boolean) => void }) {
  const progress = file.size > 0 ? Math.min(100, (file.downloaded / file.size) * 100) : 0;
  return <label className="tg-file-row"><input checked={file.selected} onChange={(event) => onChange(event.target.checked)} type="checkbox" /><FileText size={16} /><span className="tg-file-path" title={file.path}>{file.path}</span><span className="tg-number is-muted">{formatBytes(file.size)}</span><span className="tg-file-progress"><span className="tg-meter"><i style={{ width: `${progress}%` }} /></span><span>{progress.toFixed(0)}%</span></span></label>;
}

export function TorrentDetails({ torrent, onFileSelected }: TorrentDetailsProps) {
  const [tab, setTab] = useState<DetailTab>("overview");
  const [details, setDetails] = useState<TorrentDetailsModel | null>(null);
  const [detailsError, setDetailsError] = useState<string | null>(null);
  const [detailsRevision, setDetailsRevision] = useState(0);

  useEffect(() => {
    let active = true;
    if (!torrent) { setDetails(null); return () => { active = false; }; }
    void torrentService.getTorrentDetails(torrent.id).then((next) => {
      if (active) { setDetails(next); setDetailsError(null); }
    }).catch((cause: unknown) => {
      if (active) setDetailsError(cause instanceof Error ? cause.message : String(cause));
    });
    return () => { active = false; };
  }, [detailsRevision, torrent?.id, torrent?.downloaded, torrent?.uploaded]);

  if (!torrent) return <section className="tg-card tg-details"><div className="tg-details-empty">Select a torrent to inspect its live transfer information.</div></section>;
  const info = details?.info ?? torrent;
  const meterClass = info.status === "Seeding" ? "seed" : info.status === "Paused" ? "paused" : info.status === "Stopped" ? "stopped" : "";
  const tabs: Array<[DetailTab, string]> = [["overview", "Overview"], ["files", "Files"], ["peers", "Peers"], ["trackers", "Trackers"]];
  return (
    <section aria-label="Torrent details" className="tg-card tg-details">
      <div className="tg-details-head"><div className="tg-details-title" title={torrent.name}>{torrent.name}</div><div className="tg-tabs" role="tablist">{tabs.map(([id, label]) => <button aria-selected={tab === id} className={tab === id ? "is-active" : ""} key={id} onClick={() => setTab(id)} role="tab" type="button">{label}</button>)}</div></div>
      {detailsError ? <div className="tg-details-empty">{detailsError}</div> : details?.metadataPending ? <div className="tg-details-empty">Fetching torrent metadata. File, peer, and tracker details will appear when it is available.</div> : <div className="tg-detail-body">
        {tab === "overview" && <><div className="tg-detail-progress-head"><span>Progress</span><span>{info.progress.toFixed(1)}% complete</span></div><div className={`tg-meter tg-piece-meter ${meterClass}`}><i style={{ width: `${info.progress}%` }} /></div><dl className="tg-facts"><DetailItem label="Save to" value={info.downloadDirectory} /><DetailItem label="Status" value={info.status} /><DetailItem label="Size" value={formatBytes(info.size)} /><DetailItem label="Downloaded" value={formatBytes(info.downloaded)} /><DetailItem label="Uploaded" value={formatBytes(info.uploaded)} /><DetailItem label="Remaining" value={formatBytes(info.remaining)} /><DetailItem label="Download speed" value={formatRate(info.downloadRate)} /><DetailItem label="Upload speed" value={formatRate(info.uploadRate)} /><DetailItem label="ETA" value={formatETA(info.etaSeconds)} /><DetailItem label="Seeds · Peers" value={`${info.seeds} · ${info.peers}`} /></dl></>}
        {tab === "files" && <div className="tg-detail-list">{details?.files.length ? details.files.map((file) => <FileRow file={file} key={file.index} onChange={(selected) => { void onFileSelected(file.index, selected).then(() => setDetailsRevision((revision) => revision + 1)); }} />) : <p className="tg-details-empty">No file metadata is available.</p>}</div>}
        {tab === "peers" && <div className="tg-detail-list">{details?.peers.length ? <><div className="tg-peer-head"><span>Address</span><span>Client</span><span>Down</span><span>Up</span></div>{details.peers.map((peer) => <div className="tg-peer-row" key={`${peer.network}:${peer.address}`}><span className="tg-number">{peer.address}</span><span>{peer.client}</span><span className="tg-number">{formatRate(peer.downloadRate)}</span><span className="tg-number is-muted">{formatRate(peer.uploadRate)}</span></div>)}</> : <p className="tg-details-empty"><Users size={16} /> No connected peers right now.</p>}</div>}
        {tab === "trackers" && <div className="tg-detail-list">{details?.trackers.length ? details.trackers.map((tracker) => <div className="tg-tracker-row" key={`${tracker.tier}:${tracker.url}`}><code>{tracker.url}</code><span>Tier {tracker.tier}</span></div>) : <p className="tg-details-empty">No tracker URLs were supplied by this torrent. DHT and peer exchange may still discover peers.</p>}</div>}
      </div>}
    </section>
  );
}
