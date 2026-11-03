import { useCallback, useEffect, useState } from "react";
import { emptySettings, emptySnapshot, torrentService } from "../services/torrent-service";
import type { Settings, TorrentSnapshot } from "../types/torrent";

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function useTorrents() {
  const [snapshot, setSnapshot] = useState<TorrentSnapshot>(emptySnapshot);
  const [settings, setSettings] = useState<Settings>(emptySettings);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    const [nextSnapshot, nextSettings] = await Promise.all([
      torrentService.getTorrents(),
      torrentService.getSettings(),
    ]);
    setSnapshot(nextSnapshot);
    setSettings(nextSettings);
  }, []);

  useEffect(() => {
    let active = true;
    const unsubscribe = torrentService.subscribe((nextSnapshot) => {
      if (active) setSnapshot(nextSnapshot);
    });

    void refresh()
      .catch((cause: unknown) => {
        if (active) setError(errorMessage(cause));
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
      unsubscribe();
    };
  }, [refresh]);

  return { error, loading, refresh, setError, setSettings, settings, snapshot };
}
