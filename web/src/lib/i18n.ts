import i18n from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";

import de from "../locales/de.json";
import en from "../locales/en.json";
import fr from "../locales/fr.json";

const resources = {
  en: {
    translation: en,
  },
  fr: {
    translation: fr,
  },
  de: {
    translation: de,
  },
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: "en",
    debug: false,

    detection: {
      order: ["cookie", "navigator", "htmlTag"],
      caches: ["cookie"],
      cookieOptions: {
        path: "/",
        sameSite: "strict",
      },
    },

    interpolation: {
      escapeValue: false,
    },
  });

export default i18n;
