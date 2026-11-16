<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type { Copy } from "../i18n";
import NumberedTextarea from "./NumberedTextarea.vue";
import {
  copyText,
  downloadText,
  formatTracks,
  type Playlist,
  type ExportOptions,
} from "../lib/playlist";
const props = defineProps<{
  t: Copy;
  list: Playlist | null;
  busy: boolean;
  options: ExportOptions;
}>();
const batchSize = ref(0);
const batchIndex = ref(0);
const text = ref("");
const notice = ref<"copied" | "copyFailed" | "downloaded" | "">("");
const songs = computed(() =>
  formatTracks(props.list?.tracks || [], props.options),
);
const batches = computed(() =>
  batchSize.value
    ? Math.max(1, Math.ceil(songs.value.length / batchSize.value))
    : 1,
);
const start = computed(() =>
  batchSize.value ? batchIndex.value * batchSize.value : 0,
);
const current = computed(() =>
  batchSize.value
    ? songs.value.slice(start.value, start.value + batchSize.value)
    : songs.value,
);
watch([songs, batchSize], () => {
  batchIndex.value = 0;
});
watch(
  current,
  (values) => {
    text.value = values.join("\n");
    notice.value = "";
  },
  { immediate: true },
);
watch(text, () => {
  notice.value = "";
});
async function copy() {
  try {
    await copyText(text.value);
    notice.value = "copied";
  } catch {
    notice.value = "copyFailed";
  }
}
function download() {
  downloadText(
    (props.list?.name || "GoMusic") +
      (batches.value > 1 ? `-${batchIndex.value + 1}` : ""),
    text.value,
  );
  notice.value = "downloaded";
}
</script>
<template>
  <section
    class="export-panel"
    :aria-busy="busy"
    aria-labelledby="export-heading"
  >
    <div class="result-heading">
      <h2 id="export-heading">{{ t.results }}</h2>
      <span v-if="list"
        >{{ t.received }} {{ list.tracks.length }} {{ t.songs }}</span
      >
    </div>
    <p v-if="list?.name" class="playlist-name">
      {{ t.playlistName }}{{ list.name }}
    </p>
    <p v-if="list && !list.complete" class="partial-warning" role="status">
      <strong>{{ t.partial }}</strong> · {{ t.received }}
      {{ list.tracks.length }} / {{ t.total }} {{ list.total_count }}<br />{{
        list.source === "kugou" && list.preview
          ? t.kugouPreviewHint
          : t.partialHint
      }}
    </p>
    <details
      v-if="list && !list.complete && list.warnings.length"
      class="partial-details"
    >
      <summary>{{ t.warningDetails }}</summary>
      <ul>
        <li
          v-for="(warning, index) in list.warnings"
          :key="`${index}-${warning}`"
        >
          {{ warning }}
        </li>
      </ul>
    </details>
    <div
      v-if="songs.length > 500"
      class="batch-control"
      role="radiogroup"
      :aria-label="t.batch"
    >
      <span class="option-label">{{ t.batch }}</span>
      <div class="option-values">
        <label
          ><input
            v-model.number="batchSize"
            type="radio"
            name="batch-size"
            :value="0"
          />{{ t.all }}</label
        >
        <label
          ><input
            v-model.number="batchSize"
            type="radio"
            name="batch-size"
            :value="500"
          />{{ t.batch500 }}</label
        >
      </div>
    </div>
    <NumberedTextarea
      v-model="text"
      :label="t.resultLabel"
      :placeholder="busy ? t.loading : t.resultPlaceholder"
      :disabled="!list || busy"
      :start="start + 1"
    />
    <div v-if="list?.tracks.length" class="text-caption">
      <span>{{ t.editable }}</span><span
        >{{ start + 1 }}–{{ start + current.length }} / {{ songs.length }}</span
      >
    </div>
    <div v-if="batches > 1" class="batch-navigation">
      <button
        :disabled="batchIndex === 0"
        :aria-label="t.previous"
        @click="batchIndex--"
      >
        ←</button
      ><span>{{ t.batchLabel }} {{ batchIndex + 1 }} / {{ batches }}</span
      ><button
        :disabled="batchIndex + 1 >= batches"
        :aria-label="t.next"
        @click="batchIndex++"
      >
        →
      </button>
    </div>
    <p v-if="list && !list.tracks.length" class="empty-playlist">
      {{ t.emptyPlaylist }}
    </p>
    <div class="export-actions">
      <button
        class="primary-button"
        :disabled="!text.trim() || busy"
        @click="copy"
      >
        {{ t.copy }}</button><button
        class="secondary-button"
        :disabled="!text.trim() || busy"
        @click="download"
      >
        {{ t.download }}</button
      ><span v-if="notice" class="action-notice" role="status">{{
        t[notice]
      }}</span>
    </div>
  </section>
</template>
