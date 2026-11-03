import { Events } from "@wailsio/runtime";
import { TorrentgoTorrentImport, TorrentgoTorrents } from "../../bindings/events";
import {
  Settings,
  TorrentImportResult,
  TorrentSnapshot,
} from "../../bindings/github.com/nikitrajkovski/torrentgo/internal/models/models";
import {
  AddMagnet,
  AddTorrentFile,
  ChooseDownloadDirectory,
  GetSettings,
  GetTorrents,
  PauseAll,
  PauseTorrent,
  RemoveTorrent,
  ResumeAll,
  ResumeTorrent,
  StopTorrent,
  UpdateSettings,
} from "../../bindings/github.com/nikitrajkovski/torrentgo/internal/services/torrentservice";

export const torrentService = {
  addMagnet: AddMagnet,
  addTorrentFile: AddTorrentFile,
  chooseDownloadDirectory: ChooseDownloadDirectory,
  getSettings: GetSettings,
  getTorrents: GetTorrents,
  pause: PauseTorrent,
  pauseAll: PauseAll,
  remove: (id: string) => RemoveTorrent(id, false),
  resume: ResumeTorrent,
  resumeAll: ResumeAll,
  stop: StopTorrent,
  updateSettings: UpdateSettings,
  subscribe(onSnapshot: (snapshot: TorrentSnapshot) => void): () => void {
    return Events.On(TorrentgoTorrents, (event) => {
      onSnapshot(event.data);
    });
  },
  subscribeImports(onResult: (result: TorrentImportResult) => void): () => void {
    return Events.On(TorrentgoTorrentImport, (event) => {
      onResult(event.data);
    });
  },
};

export function emptySnapshot(): TorrentSnapshot {
  return new TorrentSnapshot({ torrents: [], networkReady: false });
}

export function emptySettings(): Settings {
  return new Settings({ downloadDirectory: "", theme: "dark" });
}
