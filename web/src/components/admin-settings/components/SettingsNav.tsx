import { useTranslation } from "react-i18next";

const groups: Array<{ key: string; sections: Array<string> }> = [
  { key: "core", sections: ["app", "workers"] },
  {
    key: "infrastructure",
    sections: [
      "database",
      "cache",
      "storage",
      "events",
      "notifier",
      "activity",
    ],
  },
  { key: "security", sections: ["auth", "security"] },
  { key: "observability", sections: ["observability"] },
];

const sectionLabelKey: Record<string, string> = {
  app: "admin.settings.sections.app",
  workers: "admin.settings.sections.workers",
  database: "admin.settings.sections.database",
  cache: "admin.settings.sections.cache",
  storage: "admin.settings.sections.storage",
  events: "admin.settings.sections.events",
  notifier: "admin.settings.sections.notifier",
  activity: "admin.settings.sections.activity",
  auth: "admin.settings.sections.auth",
  observability: "admin.settings.sections.observability",
  security: "admin.settings.sections.security",
};

const groupLabelKey: Record<string, string> = {
  core: "admin.settings.nav.core",
  infrastructure: "admin.settings.nav.infrastructure",
  security: "admin.settings.nav.security",
  observability: "admin.settings.nav.observability",
};

export function SettingsNav() {
  const { t } = useTranslation();

  return (
    <nav className="sticky top-6 hidden h-fit w-48 shrink-0 lg:block">
      <ul className="space-y-4 text-sm">
        {groups.map((group) => (
          <li key={group.key}>
            <div className="mb-1 px-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
              {t(groupLabelKey[group.key])}
            </div>
            <ul className="space-y-0.5">
              {group.sections.map((section) => (
                <li key={section}>
                  <a
                    href={`#${section}`}
                    className="block rounded px-2 py-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  >
                    {t(sectionLabelKey[section])}
                  </a>
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
    </nav>
  );
}
