import { CheckCircle2, CirclePause, Download, Plus, Settings2, Upload } from "lucide-react";

export type TorrentView = "all" | "downloading" | "seeding" | "completed" | "paused" | "settings";
type TorrentListView = Exclude<TorrentView, "settings">;

type SidebarProps = {
  activeView: TorrentView;
  counts: Record<TorrentListView, number>;
  onAdd: () => void;
  onSelect: (view: TorrentView) => void;
};

const items: Array<{ id: TorrentListView; label: string; icon: typeof Download }> = [
  { id: "all", label: "All torrents", icon: Download },
  { id: "downloading", label: "Downloading", icon: Download },
  { id: "seeding", label: "Seeding", icon: Upload },
  { id: "completed", label: "Completed", icon: CheckCircle2 },
  { id: "paused", label: "Paused", icon: CirclePause },
];

export function Sidebar({ activeView, counts, onAdd, onSelect }: SidebarProps) {
  return (
    <aside className="tg-sidebar">
      <button className="tg-primary" onClick={onAdd} type="button"><Plus size={18} strokeWidth={2.25} /> Add torrent</button>
      <nav aria-label="Library filters">
        <div className="tg-nav-label">Library</div>
        <div className="tg-nav">
          {items.map(({ id, icon: Icon, label }) => (
            <button aria-pressed={activeView === id} className={`tg-nav-button ${activeView === id ? "is-active" : ""}`} key={id} onClick={() => onSelect(id)} type="button">
              <Icon size={17} /><span>{label}</span><span className="tg-nav-count">{counts[id]}</span>
            </button>
          ))}
        </div>
      </nav>
      <div className="tg-sidebar-spacer" />
      <p className="tg-side-note">TorrentGo runs locally. No ads, telemetry, or cloud downloads.</p>
      <button className={`tg-settings-link ${activeView === "settings" ? "is-active" : ""}`} onClick={() => onSelect("settings")} type="button"><Settings2 size={17} /> Settings</button>
    </aside>
  );
}
