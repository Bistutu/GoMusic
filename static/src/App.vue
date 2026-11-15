<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { messages } from "./i18n";
import { usePlaylist } from "./composables/usePlaylist";
import type { ExportOptions } from "./lib/playlist";
import PlaylistForm from "./components/PlaylistForm.vue";
import ExportPanel from "./components/ExportPanel.vue";
import TransferGuide from "./components/TransferGuide.vue";
function preference(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}
const language = ref<"zh" | "en">(
  preference("gomusic-language") === "en" ? "en" : "zh",
);
const exportOptions = reactive<ExportOptions>({
  detailed: false,
  format: "song-singer",
  order: "normal",
});
const showStars = ref(true);
const starsLoaded = ref(false);
const t = computed(() => messages[language.value]);
watch(
  language,
  (value) => {
    document.documentElement.lang = value === "zh" ? "zh-CN" : "en";
    try {
      localStorage.setItem("gomusic-language", value);
    } catch {
      /* Optional preference. */
    }
  },
  { immediate: true },
);
const { link, playlist, loading, error, load, cancel } = usePlaylist();
const errorMessage = computed(() => {
  if (!error.value) return "";
  const keys = {
    invalid: "errorInvalid",
    network: "errorNetwork",
    timeout: "errorTimeout",
    server: "errorServer",
    response: "errorResponse",
  } as const;
  return t.value[keys[error.value.kind]];
});
</script>
<template>
  <div class="site-shell">
    <header class="site-header">
      <a class="brand" href="/" aria-label="GoMusic">GoMusic</a>
      <nav aria-label="GoMusic">
        <a
          href="https://github.com/Bistutu/GoMusic"
          target="_blank"
          rel="noopener noreferrer"
          ><img
            v-if="showStars"
            v-show="starsLoaded"
            class="github-stars"
            src="https://img.shields.io/github/stars/Bistutu/GoMusic?style=flat-square&logo=github&label=Star"
            alt="GitHub stars"
            height="20"
            @load="starsLoaded = true"
            @error="showStars = false"
          /><span v-if="!starsLoaded || !showStars">GitHub</span></a
        >
        <button
          class="text-button"
          :aria-label="t.language"
          @click="language = language === 'zh' ? 'en' : 'zh'"
        >
          {{ language === "zh" ? "EN" : "中文" }}
        </button>
      </nav>
    </header>
    <main>
      <div class="page-intro">
        <h1>
          <template v-for="(part, index) in t.title.split(' / ')" :key="index"
            ><template v-if="index"> / </template
            ><span class="title-part">{{ part }}</span></template
          >
        </h1>
        <p>{{ t.subtitle }}</p>
      </div>
      <PlaylistForm
        v-model="link"
        :t="t"
        :busy="loading"
        :error="errorMessage"
        :options="exportOptions"
        @submit="load"
        @cancel="cancel"
      />
      <ExportPanel
        :t="t"
        :list="playlist"
        :busy="loading"
        :options="exportOptions"
      />
      <TransferGuide :t="t" />
    </main>
  </div>
</template>
