import { defineConfig } from "i18next-cli";

export default defineConfig({
  locales: ["en", "fr"],

  extract: {
    input: ["src/**/*.{ts,tsx}"],
    output: "src/locales/{{language}}.json",
    // Flat output (no namespace in path, keys are nested in single file)
    defaultNS: "translation",
    keySeparator: ".",
    nsSeparator: false,
    // Don't auto-remove unused keys without explicit review
    removeUnusedKeys: false,
  },
});
