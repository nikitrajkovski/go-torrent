import { Events } from "@wailsio/runtime";
import { TorrentgoTorrents } from "../../bindings/events";
import {
  Settings,
  TorrentSnapshot,
} from "../../bindings/github.com/nikitrajkovski/torrentgo/internal/models/models";
import {
  AddMagnet,
  AddTorrentFile,
  ChooseDownloadDirectory,
  GetSettings,
  GetTorrents,
  PauseTorrent,
  RemoveTorrent,
  ResumeTorrent,
} from "../../bindings/github.com/nikitrajkovski/torrentgo/internal/services/torrentservice";

export const torrentService = {
  addMagnet: AddMagnet,
  addTorrentFile: AddTorrentFile,
  chooseDownloadDirectory: ChooseDownloadDirectory,
  getSettings: GetSettings,
  getTorrents: GetTorrents,
  pause: PauseTorrent,
  remove: (id: string) => RemoveTorrent(id, false),
  resume: ResumeTorrent,
  subscribe(onSnapshot: (snapshot: TorrentSnapshot) => void): () => void {
    return Events.On(TorrentgoTorrents, (event) => {
      onSnapshot(event.data);
    });
  },
};

export function emptySnapshot(): TorrentSnapshot {
  return new TorrentSnapshot({ torrents: [], networkReady: false });
}

export function emptySettings(): Settings {
  return new Settings({ downloadDirectory: "" });
}
