export type ColorTheme =
  | "default"
  | "darkmatter"
  | "quantum_rose"
  | "ocean_breeze"
  | "elegant_luxury"
  | "neo_brutalism";

export interface ThemeColors {
  primary: string;
  secondary: string;
  accent: string;
  muted: string;
}

export interface ThemeConfig {
  name: ColorTheme;
  label: string;
  cssPath: string;
  previewColors: {
    light: ThemeColors;
    dark: ThemeColors;
  };
}

export const themes: Record<ColorTheme, ThemeConfig> = {
  default: {
    name: "default",
    label: "Default",
    cssPath: "/src/styles/themes/default.css",
    previewColors: {
      light: {
        primary: "oklch(0.5417 0.179 288.0332)",
        secondary: "oklch(0.9174 0.0435 292.6901)",
        accent: "oklch(0.9221 0.0373 262.141)",
        muted: "oklch(0.958 0.0133 286.1454)",
      },
      dark: {
        primary: "oklch(0.7162 0.1597 290.3962)",
        secondary: "oklch(0.3139 0.0736 283.4591)",
        accent: "oklch(0.3354 0.0828 280.9705)",
        muted: "oklch(0.271 0.0621 281.4377)",
      },
    },
  },
  darkmatter: {
    name: "darkmatter",
    label: "Darkmatter",
    cssPath: "/src/styles/themes/darkmatter.css",
    previewColors: {
      light: {
        primary: "oklch(0.6716 0.1368 48.5130)",
        secondary: "oklch(0.5360 0.0398 196.0280)",
        accent: "oklch(0.9491 0 0)",
        muted: "oklch(0.9670 0.0029 264.5419)",
      },
      dark: {
        primary: "oklch(0.7214 0.1337 49.9802)",
        secondary: "oklch(0.5940 0.0443 196.0233)",
        accent: "oklch(0.3211 0 0)",
        muted: "oklch(0.2520 0 0)",
      },
    },
  },
  quantum_rose: {
    name: "quantum_rose",
    label: "Quantum Rose",
    cssPath: "/src/styles/themes/quantum_rose.css",
    previewColors: {
      light: {
        primary: "oklch(0.6002 0.2414 0.1348)",
        secondary: "oklch(0.9230 0.0701 326.1273)",
        accent: "oklch(0.8766 0.0828 344.8849)",
        muted: "oklch(0.9429 0.0363 344.2604)",
      },
      dark: {
        primary: "oklch(0.7543 0.2319 332.0212)",
        secondary: "oklch(0.3184 0.0915 319.6465)",
        accent: "oklch(0.3558 0.1201 325.7655)",
        muted: "oklch(0.2701 0.0770 312.3525)",
      },
    },
  },
  ocean_breeze: {
    name: "ocean_breeze",
    label: "Ocean Breeze",
    cssPath: "/src/styles/themes/ocean_breeze.css",
    previewColors: {
      light: {
        primary: "oklch(0.7227 0.1920 149.5793)",
        secondary: "oklch(0.9514 0.0250 236.8242)",
        accent: "oklch(0.9505 0.0507 163.0508)",
        muted: "oklch(0.9670 0.0029 264.5419)",
      },
      dark: {
        primary: "oklch(0.7729 0.1535 163.2231)",
        secondary: "oklch(0.3351 0.0331 260.9120)",
        accent: "oklch(0.3729 0.0306 259.7328)",
        muted: "oklch(0.2463 0.0275 259.9628)",
      },
    },
  },
  elegant_luxury: {
    name: "elegant_luxury",
    label: "Elegant Luxury",
    cssPath: "/src/styles/themes/elegant_luxury.css",
    previewColors: {
      light: {
        primary: "oklch(0.4650 0.1470 24.9381)",
        secondary: "oklch(0.9625 0.0385 89.0943)",
        accent: "oklch(0.9619 0.0580 95.6174)",
        muted: "oklch(0.9431 0.0068 53.4442)",
      },
      dark: {
        primary: "oklch(0.5054 0.1905 27.5181)",
        secondary: "oklch(0.4732 0.1247 46.2007)",
        accent: "oklch(0.5553 0.1455 48.9975)",
        muted: "oklch(0.2291 0.0060 56.0708)",
      },
    },
  },
  neo_brutalism: {
    name: "neo_brutalism",
    label: "Neo Brutalism",
    cssPath: "/src/styles/themes/neo_brutalism.css",
    previewColors: {
      light: {
        primary: "oklch(0.6489 0.2370 26.9728)",
        secondary: "oklch(0.9680 0.2110 109.7692)",
        accent: "oklch(0.5635 0.2408 260.8178)",
        muted: "oklch(0.9551 0 0)",
      },
      dark: {
        primary: "oklch(0.7044 0.1872 23.1858)",
        secondary: "oklch(0.9691 0.2005 109.6228)",
        accent: "oklch(0.6755 0.1765 252.2592)",
        muted: "oklch(0.2178 0 0)",
      },
    },
  },
};

let currentThemeLink: HTMLLinkElement | null = null;

export async function loadTheme(theme: ColorTheme) {
  const themeConfig = themes[theme];

  if (currentThemeLink) {
    currentThemeLink.remove();
    currentThemeLink = null;
  }

  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = themeConfig.cssPath;
  link.setAttribute("data-theme", theme);
  document.head.appendChild(link);

  currentThemeLink = link;

  return new Promise<void>((resolve, reject) => {
    link.onload = () => resolve();
    link.onerror = () => reject(new Error(`Failed to load theme: ${theme}`));
  });
}
