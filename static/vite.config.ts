import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: process.env.GOMUSIC_API_PROXY || "http://127.0.0.1:8081",
      },
    },
  },
  build: { target: "es2020" },
  test: { include: ["src/**/*.test.ts"], environment: "node" },
});
