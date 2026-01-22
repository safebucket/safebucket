//  @ts-check
import { tanstackConfig } from "@tanstack/eslint-config";
import i18next from "eslint-plugin-i18next";
import jsxA11y from "eslint-plugin-jsx-a11y";

export default [
  {
    ignores: ["eslint.config.js"],
  },
  ...tanstackConfig,
  jsxA11y.flatConfigs.recommended,
  i18next.configs["flat/recommended"],
  {
    rules: {
      // Set i18next to warn initially - flip to "error" once translations are complete
      "i18next/no-literal-string": "warn",
      // Relaxed rules for existing code - tighten these over time
      "@typescript-eslint/no-unnecessary-condition": "warn",
      "jsx-a11y/click-events-have-key-events": "warn",
      "jsx-a11y/no-static-element-interactions": "warn",
      "jsx-a11y/no-autofocus": "warn",
    },
  },
];
