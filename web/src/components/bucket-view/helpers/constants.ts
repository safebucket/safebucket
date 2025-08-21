export const shareFileFields = [
  { id: "files", label: "File", type: "file" as const, required: true },
  // TODO(YLB): Uncomment when needed
  // {
  //   id: "password",
  //   label: "Password",
  //   type: "text" as const,
  //   defaultValue: "0UymxETG$wc)7k8",
  // },
  // {
  //   id: "maxDownloads",
  //   label: "Max downloads",
  //   type: "select" as const,
  //   options: [
  //     { value: "unlimited", label: "Unlimited" },
  //     { value: "1", label: "1" },
  //     { value: "3", label: "3" },
  //     { value: "5", label: "5" },
  //   ],
  //   defaultValue: "unlimited",
  // },
  // {
  //   id: "expiresAt",
  //   label: "Expires at",
  //   type: "switch" as const,
  //   defaultValue: false,
  // },
  // {
  //   id: "expiresAtDate",
  //   label: "Date",
  //   type: "datepicker" as const,
  //   condition: (values: FieldValues) => !!values.expiresAt,
  // },
];
