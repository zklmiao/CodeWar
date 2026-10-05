import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      "/ws": { target: "ws://localhost:3001", ws: true },
      "/api": "http://localhost:3001",
    },
  },
});
