import { fileURLToPath, URL } from "node:url";
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    // design/ 与 client/ 平级：dev server 放行仓库根（tokens.css 在 client 之外）
    fs: { allow: [fileURLToPath(new URL("../..", import.meta.url))] },
    proxy: {
      "/ws": { target: "ws://localhost:3001", ws: true },
      "/api": "http://localhost:3001",
    },
  },
});
