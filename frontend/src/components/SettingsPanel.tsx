import { useEffect, useState } from "react";
import { FolderOpen, Moon, Sun } from "lucide-react";
import type { Settings } from "../types/torrent";

type SettingsPanelProps = {
  busy: boolean;
  onChooseDirectory: () => void;
  onThemeChange: (theme: "dark" | "light") => void;
  onTransferSettings: (downloadKiB: number, uploadKiB: number, concurrent: number) => void;
  settings: Settings;
};

export function SettingsPanel({ busy, onChooseDirectory, onThemeChange, onTransferSettings, settings }: SettingsPanelProps) {
  const [downloadKiB, setDownloadKiB] = useState(String(Math.round(settings.downloadLimit / 1024)));
  const [uploadKiB, setUploadKiB] = useState(String(Math.round(settings.uploadLimit / 1024)));
  const [concurrent, setConcurrent] = useState(String(settings.maxConcurrentDownloads));
  useEffect(() => {
    setDownloadKiB(String(Math.round(settings.downloadLimit / 1024)));
    setUploadKiB(String(Math.round(settings.uploadLimit / 1024)));
    setConcurrent(String(settings.maxConcurrentDownloads));
  }, [settings.downloadLimit, settings.maxConcurrentDownloads, settings.uploadLimit]);
  const saveTransfers = (event: React.FormEvent) => {
    event.preventDefault();
    onTransferSettings(Number(downloadKiB), Number(uploadKiB), Number(concurrent));
  };
  return (
    <section className="tg-card tg-settings">
      <h2>Settings</h2>
      <div className="tg-settings-row"><div><div className="tg-settings-label">Download destination</div><div className="tg-settings-description">{settings.downloadDirectory}</div></div><button className="tg-secondary" disabled={busy} onClick={onChooseDirectory} type="button"><FolderOpen size={16} /> Choose folder</button></div>
      <div className="tg-settings-row"><div><div className="tg-settings-label">Appearance</div><div className="tg-settings-description">Choose the local application theme.</div></div><div className="tg-theme-switch"><button aria-pressed={settings.theme === "dark"} className={settings.theme === "dark" ? "is-active" : ""} disabled={busy} onClick={() => onThemeChange("dark")} type="button"><Moon size={14} /> Dark</button><button aria-pressed={settings.theme === "light"} className={settings.theme === "light" ? "is-active" : ""} disabled={busy} onClick={() => onThemeChange("light")} type="button"><Sun size={14} /> Light</button></div></div>
      <form className="tg-transfer-settings" onSubmit={saveTransfers}>
        <div><div className="tg-settings-label">Transfer controls</div><div className="tg-settings-description">Use 0 for unlimited speed or unlimited concurrent downloads.</div></div>
        <label>Download limit KiB/s<input min="0" onChange={(event) => setDownloadKiB(event.target.value)} type="number" value={downloadKiB} /></label>
        <label>Upload limit KiB/s<input min="0" onChange={(event) => setUploadKiB(event.target.value)} type="number" value={uploadKiB} /></label>
        <label>Concurrent downloads<input max="64" min="0" onChange={(event) => setConcurrent(event.target.value)} type="number" value={concurrent} /></label>
        <button className="tg-primary" disabled={busy} type="submit">Save transfer controls</button>
      </form>
    </section>
  );
}
