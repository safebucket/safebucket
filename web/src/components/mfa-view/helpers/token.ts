const KEY = "mfa_pending";

export const mfaPending = {
  get: () => sessionStorage.getItem(KEY) === "1",
  set: () => sessionStorage.setItem(KEY, "1"),
  clear: () => sessionStorage.removeItem(KEY),
};
