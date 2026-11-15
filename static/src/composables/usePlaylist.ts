import { onScopeDispose, ref, shallowRef } from "vue";
import { fetchPlaylist, PlaylistError } from "../lib/api";
import { platformOf, type Playlist } from "../lib/playlist";

export function usePlaylist() {
  const link = ref("");
  const playlist = shallowRef<Playlist | null>(null);
  const loading = ref(false);
  const error = shallowRef<PlaylistError | null>(null);
  let controller: AbortController | undefined;
  let generation = 0;
  async function load() {
    if (loading.value) return;
    error.value = null;
    if (
      !platformOf(link.value) ||
      new TextEncoder().encode(link.value).length > 4096
    ) {
      error.value = new PlaylistError("invalid");
      return;
    }
    const current = ++generation;
    controller = new AbortController();
    loading.value = true;
    playlist.value = null;
    try {
      const result = await fetchPlaylist(link.value.trim(), controller.signal);
      if (current === generation) playlist.value = result;
    } catch (e) {
      if (current === generation && e instanceof PlaylistError) error.value = e;
    } finally {
      if (current === generation) loading.value = false;
    }
  }
  function cancel() {
    ++generation;
    controller?.abort();
    loading.value = false;
  }
  onScopeDispose(cancel);
  return { link, playlist, loading, error, load, cancel };
}
