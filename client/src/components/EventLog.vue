<script setup lang="ts">
import { useMatchStore } from "../stores/match";

const store = useMatchStore();
</script>

<template>
  <div class="log">
    <div v-for="(l, i) in [...store.log].reverse()" :key="i" class="line">
      <span class="seq">{{ l.seq }}</span>{{ l.text }}
    </div>
  </div>
</template>

<style scoped>
.log {
  height: 150px; overflow-y: auto; background: #141926;
  border-radius: 8px; padding: 8px 12px; font-size: 13px;
  color: #aab3c7;
}
.line { padding: 1px 0; }
.seq { display: inline-block; width: 34px; color: #56607a; }
</style>
