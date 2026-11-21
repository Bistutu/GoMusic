<script setup lang="ts">
import { computed } from "vue";
import { platformOf, type ExportOptions } from "../lib/playlist";
import type { Copy } from "../i18n";
const link = defineModel<string>({ required: true });
const props = defineProps<{
  t: Copy;
  busy: boolean;
  error: string;
  options: ExportOptions;
}>();
const emit = defineEmits<{ submit: []; cancel: [] }>();
const exampleUrl = "http://163cn.tv/zoIxm3";
const platform = computed(() => {
  const source = platformOf(link.value);
  return source ? props.t[source] : "";
});
</script>
<template>
  <form class="playlist-form" @submit.prevent="emit('submit')">
    <label class="input-label" for="playlist-link">{{ t.inputLabel }}</label>
    <textarea
      id="playlist-link"
      v-model="link"
      class="link-input"
      rows="2"
      :placeholder="t.inputPlaceholder"
      :disabled="busy"
      :aria-invalid="Boolean(error)"
      :aria-describedby="error ? 'input-error' : undefined"
      spellcheck="false"
      @keydown.enter.exact.prevent="emit('submit')"
      @keydown.ctrl.enter.prevent="emit('submit')"
      @keydown.meta.enter.prevent="emit('submit')"
    />
    <div class="input-helper">
      <span v-if="platform">{{ t.detected + platform }}</span>
      <button
        type="button"
        class="text-button example-link"
        :disabled="busy"
        :title="exampleUrl"
        @click="link = exampleUrl"
      >
        {{ t.example }}
      </button>
    </div>
    <div class="playlist-options">
      <div class="option-row" role="radiogroup" :aria-label="t.format">
        <span class="option-label">{{ t.format }}</span>
        <div class="option-values">
          <label
            ><input
              v-model="options.format"
              type="radio"
              value="song-singer"
              name="format"
            />{{ t.songSinger }}</label
          ><label
            ><input
              v-model="options.format"
              type="radio"
              value="singer-song"
              name="format"
            />{{ t.singerSong }}</label
          ><label
            ><input
              v-model="options.format"
              type="radio"
              value="song"
              name="format"
            />{{ t.songOnly }}</label
          >
        </div>
      </div>
      <div class="option-row" role="radiogroup" :aria-label="t.order">
        <span class="option-label">{{ t.order }}</span>
        <div class="option-values">
          <label
            ><input
              v-model="options.order"
              type="radio"
              value="normal"
              name="order"
            />{{ t.normal }}</label
          ><label
            ><input
              v-model="options.order"
              type="radio"
              value="reverse"
              name="order"
            />{{ t.reverse }}</label
          >
        </div>
      </div>
      <label class="original-option" :title="t.originalHint"
        ><input v-model="options.detailed" type="checkbox" />{{
          t.original
        }}</label
      >
    </div>
    <div class="fetch-actions">
      <button class="primary-button" :disabled="busy" type="submit">
        <span v-if="busy" class="spinner" />{{
          busy ? t.loading : t.fetch
        }}</button
      ><button
        v-if="busy"
        type="button"
        class="text-button"
        @click="emit('cancel')"
      >
        {{ t.cancel }}</button
      ><span v-if="busy" class="loading-note" role="status">{{
        t.loadingDescription
      }}</span>
    </div>
    <p v-if="error" id="input-error" class="error-message" role="alert">
      {{ error }}
    </p>
  </form>
</template>
