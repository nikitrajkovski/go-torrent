import { FolderOpen, Moon, Sun } from "lucide-react";
import type { Settings } from "../types/torrent";

type SettingsPanelProps = { busy: boolean; onChooseDirectory: () => void; onThemeChange: (theme: "dark" | "light") => void; settings: Settings };

export function SettingsPanel({ busy, onChooseDirectory, onThemeChange, settings }: SettingsPanelProps) {
  return (
    <section className="tg-card tg-settings">
      <h2>Settings</h2>
      <div className="tg-settings-row"><div><div className="tg-settings-label">Download destination</div><div className="tg-settings-description">{settings.downloadDirectory}</div></div><button className="tg-secondary" disabled={busy} onClick={onChooseDirectory} type="button"><FolderOpen size={16} /> Choose folder</button></div>
      <div className="tg-settings-row"><div><div className="tg-settings-label">Appearance</div><div className="tg-settings-description">Choose the local application theme.</div></div><div className="tg-theme-switch"><button aria-pressed={settings.theme === "dark"} className={settings.theme === "dark" ? "is-active" : ""} disabled={busy} onClick={() => onThemeChange("dark")} type="button"><Moon size={14} /> Dark</button><button aria-pressed={settings.theme === "light"} className={settings.theme === "light" ? "is-active" : ""} disabled={busy} onClick={() => onThemeChange("light")} type="button"><Sun size={14} /> Light</button></div></div>
    </section>
  );
}
