import { defineConfig } from "vite"
import vue from "@vitejs/plugin-vue"
import path from "node:path"
import { quasar, transformAssetUrls } from "@quasar/vite-plugin"

const dirName = import.meta.dirname

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue({
      template: { transformAssetUrls },
    }),
    quasar({
      sassVariables: "@/styles/quasar-variables.scss",
    }),
  ],
  define: { "process.env": {} },
  resolve: {
    alias: {
      "@": path.resolve(dirName, "./src"),
    },
  },
  server: {
    port: 5176,
  },
  /* remove the need to specify .vue files https://vitejs.dev/config/#resolve-extensions
  resolve: {
    extensions: [
      '.js',
      '.json',
      '.jsx',
      '.mjs',
      '.ts',
      '.tsx',
      '.vue',
    ]
  },
  */
})
