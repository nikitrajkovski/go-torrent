import { Events } from "@wailsio/runtime";
import { TorrentgoTorrentImport, TorrentgoTorrents } from "../../bindings/events";
import {
  Settings,
  TorrentDetails,
  TorrentImportResult,
  TorrentSnapshot,
} from "../../bindings/github.com/nikitrajkovski/torrentgo/internal/models/models";
import {
  AddMagnet,
  AddTorrentFile,
  ChooseDownloadDirectory,
  GetSettings,
  GetTorrentDetails,
  GetTorrents,
  OpenDownloadDirectory,
  PauseAll,
  PauseTorrent,
  RemoveTorrent,
  ResumeAll,
  ResumeTorrent,
  StopTorrent,
  StartPreparedTorrent,
  SetFileSelected,
  UpdateSettings,
} from "../../bindings/github.com/nikitrajkovski/torrentgo/internal/services/torrentservice";

export const torrentService = {
  addMagnet: AddMagnet,
  addTorrentFile: AddTorrentFile,
  chooseDownloadDirectory: ChooseDownloadDirectory,
  getSettings: GetSettings,
  getTorrentDetails: GetTorrentDetails,
  getTorrents: GetTorrents,
  openDownloadDirectory: OpenDownloadDirectory,
  pause: PauseTorrent,
  pauseAll: PauseAll,
  remove: (id: string) => RemoveTorrent(id, false),
  resume: ResumeTorrent,
  resumeAll: ResumeAll,
  stop: StopTorrent,
  startPrepared: StartPreparedTorrent,
  setFileSelected: SetFileSelected,
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
