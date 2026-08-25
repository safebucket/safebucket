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
      "@typescript-eslint/no-unnecessary-type-assertion": "off",
      "jsx-a11y/click-events-have-key-events": "warn",
      "jsx-a11y/no-static-element-interactions": "warn",
      "jsx-a11y/no-autofocus": "warn",
    },
  },
  {
    files: ["src/**/*.{ts,tsx}"],
    ignores: ["src/components/ui/**"],
    rules: {
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "Literal[value=/(^|[\\s\"'])(text|bg|border|ring|fill|stroke)-(red|green|blue|yellow|amber|emerald|orange|lime|teal|cyan|sky|indigo|violet|purple|fuchsia|pink|rose|gray|slate|zinc)-[0-9]+/]",
          message:
            "Raw Tailwind palette colors are banned outside components/ui. Use semantic tokens instead: success, warning, info, destructive, primary, secondary, muted.",
        },
        {
          selector:
            "Literal[value=/(text|bg|border|ring)-\\[#[0-9a-fA-F]{3,8}\\]/]",
          message:
            "Arbitrary color values are banned outside components/ui. Add a semantic token to styles.css instead.",
        },
      ],
    },
  },
];
