import dns from "node:dns";
import http from "node:http";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// Force Node.js to resolve DNS as IPv4-first, avoiding IPv6 proxy issues
// when Chrome on Windows connects via ::1
dns.setDefaultResultOrder("ipv4first");

const apiTarget = process.env.VITE_API_TARGET || "http://127.0.0.1:8080";

// Force proxy connections over IPv4
const ipv4Agent = new http.Agent({ family: 4 });

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      "/api": {
        target: apiTarget,
        changeOrigin: true,
        agent: ipv4Agent,
      },
      "/graphql": {
        target: apiTarget,
        changeOrigin: true,
        agent: ipv4Agent,
      },
    },
  },
});
