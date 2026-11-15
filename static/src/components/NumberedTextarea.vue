<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

const text = defineModel<string>({ required: true });
const props = withDefaults(
  defineProps<{
    label: string;
    placeholder: string;
    disabled: boolean;
    start?: number;
  }>(),
  { start: 1 },
);
const area = ref<HTMLTextAreaElement>();
const mirror = ref<HTMLElement>();
const showNumbers = computed(() => !props.disabled || text.value.length > 0);
const lines = computed(() =>
  showNumbers.value ? text.value.split(/\r\n?|\n/) : [],
);
const digits = computed(() =>
  Math.max(2, String(props.start + lines.value.length - 1).length),
);
let observer: ResizeObserver | undefined;
let frame = 0;

function syncScroll() {
  if (area.value && mirror.value)
    mirror.value.style.transform = `translateY(${-area.value.scrollTop}px)`;
}
function syncLayout() {
  if (!area.value || !mirror.value) return;
  const element = area.value;
  const style = getComputedStyle(element);
  // Match the native content width, including classic scrollbar space and
  // fractional gutter widths, so wrapped lines keep their real row heights.
  mirror.value.style.width = `${
    element.getBoundingClientRect().width +
    parseFloat(style.marginLeft) -
    (element.offsetWidth - element.clientWidth)
  }px`;
  syncScroll();
}
function queueLayout() {
  cancelAnimationFrame(frame);
  frame = requestAnimationFrame(syncLayout);
}
watch([text, () => props.start, () => props.disabled], queueLayout, {
  flush: "post",
});
onMounted(() => {
  observer = new ResizeObserver(queueLayout);
  if (area.value) observer.observe(area.value);
  window.addEventListener("resize", queueLayout);
  queueLayout();
});
onBeforeUnmount(() => {
  observer?.disconnect();
  window.removeEventListener("resize", queueLayout);
  cancelAnimationFrame(frame);
});
</script>
<template>
  <div
    class="numbered-editor"
    :class="{ 'has-line-numbers': showNumbers, 'is-disabled': disabled }"
    :style="{ '--line-number-digits': digits }"
    @click.self="area?.focus()"
  >
    <div ref="mirror" class="editor-lines" aria-hidden="true">
      <div
        v-for="(line, index) in lines"
        :key="index"
        class="editor-line"
        :data-line="start + index"
      >
        {{ line || "\u200b" }}
      </div>
    </div>
    <textarea
      ref="area"
      v-model="text"
      class="result-text"
      :aria-label="label"
      :placeholder="placeholder"
      :disabled="disabled"
      spellcheck="false"
      @scroll="syncScroll"
    />
  </div>
</template>
