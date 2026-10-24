<script setup lang="ts">
// CodeMirror 6 封装：受控 v-model + 按 resetKey（题 ID）重置内容
import { basicSetup } from "codemirror";
import { python } from "@codemirror/lang-python";
import { EditorView } from "codemirror";
import { onMounted, ref, watch } from "vue";

const props = defineProps<{ modelValue: string; resetKey: string }>();
const emit = defineEmits<{ (e: "update:modelValue", v: string): void }>();
const host = ref<HTMLElement>();
let view: EditorView | null = null;

onMounted(() => {
  view = new EditorView({
    doc: props.modelValue,
    extensions: [
      basicSetup,
      python(),
      EditorView.updateListener.of((u) => {
        if (u.docChanged) emit("update:modelValue", u.state.doc.toString());
      }),
    ],
    parent: host.value!,
  });
});

watch(
  () => props.resetKey,
  () => {
    if (!view) return;
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: props.modelValue } });
  },
);

watch(
  () => props.modelValue,
  (v) => {
    if (view && v !== view.state.doc.toString()) {
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: v } });
    }
  },
);
</script>

<template>
  <div ref="host" class="editor"></div>
</template>

<style scoped>
.editor {
  /* 22 号票：编辑器语境接 --cw-font-mono；字号 13px 归入 sm 档 */
  border: 1px solid var(--cw-color-border);
  border-radius: 8px;
  overflow: hidden;
  font-family: var(--cw-font-mono);
  font-size: var(--cw-font-size-sm);
}
</style>
