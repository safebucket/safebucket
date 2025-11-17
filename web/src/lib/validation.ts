/**
 * File and folder name validation utilities for S3 storage
 * Matches backend validation rules in internal/middlewares/validator.go
 */

import type { TFunction } from "i18next";

// Allowed special characters: _ - ( ) .
const ALLOWED_SPECIAL_CHARS = new Set(["_", "-", "(", ")", ".", " "]);

export interface ValidationResult {
  valid: boolean;
  error?: string;
}

/**
 * Validates a filename according to SafeBucket rules for S3 storage
 * @param name - The filename to validate
 * @param t - Translation function from i18next
 * @param maxLength - Maximum length allowed (default: 255)
 * @returns ValidationResult with valid flag and optional error message
 */
export function validateFileName(
  name: string,
  t: TFunction,
  maxLength: number = 255,
): ValidationResult {
  const normalized = name.normalize("NFC");

  if (normalized.trim() === "") {
    return { valid: false, error: t("validation.name_empty") };
  }

  if (normalized.length > maxLength) {
    return {
      valid: false,
      error: t("validation.name_too_long", { maxLength }),
    };
  }

  if (normalized.includes("..")) {
    return { valid: false, error: t("validation.path_traversal") };
  }

  for (let i = 0; i < normalized.length; i++) {
    const char = normalized[i];
    const code = char.charCodeAt(0);

    // Block control characters (0x00-0x1F, 0x7F-0x9F)
    if (code < 0x20 || (code >= 0x7f && code <= 0x9f)) {
      return { valid: false, error: t("validation.control_characters") };
    }

    // Block path separators
    if (char === "/" || char === "\\") {
      return { valid: false, error: t("validation.path_separators") };
    }

    // Block % to prevent URL encoding confusion
    if (char === "%") {
      return {
        valid: false,
        error: t("validation.invalid_character", { char }),
      };
    }

    // Check if character is allowed
    const isLetter = /\p{L}/u.test(char);
    const isNumber = /\p{N}/u.test(char);
    const isAllowedSpecial = ALLOWED_SPECIAL_CHARS.has(char);

    if (!isLetter && !isNumber && !isAllowedSpecial) {
      return {
        valid: false,
        error: t("validation.invalid_character", { char }),
      };
    }
  }

  return { valid: true };
}

/**
 * Validates a folder name according to SafeBucket rules for S3 storage
 * @param name - The folder name to validate
 * @param t - Translation function from i18next
 * @param maxLength - Maximum length allowed (default: 255)
 * @returns ValidationResult with valid flag and optional error message
 */
export function validateFolderName(
  name: string,
  t: TFunction,
  maxLength: number = 255,
): ValidationResult {
  // Use same validation logic as files (no extension requirement for S3)
  return validateFileName(name, t, maxLength);
}

/**
 * Validates a bucket name according to SafeBucket rules for S3 storage
 * @param name - The bucket name to validate
 * @param t - Translation function from i18next
 * @param maxLength - Maximum length allowed (default: 100)
 * @returns ValidationResult with valid flag and optional error message
 */
export function validateBucketName(
  name: string,
  t: TFunction,
  maxLength: number = 100,
): ValidationResult {
  return validateFileName(name, t, maxLength);
}

/**
 * Validates a path for S3 storage
 * Allows forward slashes for directory structure but applies same character rules as filenames
 * @param path - The path to validate
 * @param t - Translation function from i18next
 * @param maxLength - Maximum length allowed (default: 1024)
 * @returns ValidationResult with valid flag and optional error message
 */
export function validatePath(
  path: string,
  t: TFunction,
  maxLength: number = 1024,
): ValidationResult {
  const normalized = path.normalize("NFC");

  // Empty path or just "/" is valid
  if (normalized === "" || normalized === "/") {
    return { valid: true };
  }

  if (normalized.length > maxLength) {
    return {
      valid: false,
      error: t("validation.path_too_long", { maxLength }),
    };
  }

  if (normalized.includes("..")) {
    return { valid: false, error: t("validation.path_traversal") };
  }

  // Split by / and validate each segment
  const segments = normalized.split("/");
  for (const segment of segments) {
    // Empty segments are OK (from leading/trailing slashes)
    if (segment === "") {
      continue;
    }

    // Validate each segment using same rules as filename (but without the slash check)
    for (let i = 0; i < segment.length; i++) {
      const char = segment[i];
      const code = char.charCodeAt(0);

      // Block control characters (0x00-0x1F, 0x7F-0x9F)
      if (code < 0x20 || (code >= 0x7f && code <= 0x9f)) {
        return { valid: false, error: t("validation.control_characters") };
      }

      // Block backslash
      if (char === "\\") {
        return { valid: false, error: t("validation.path_separators") };
      }

      // Block % to prevent URL encoding confusion
      if (char === "%") {
        return {
          valid: false,
          error: t("validation.invalid_character", { char }),
        };
      }

      // Check if character is allowed
      const isLetter = /\p{L}/u.test(char);
      const isNumber = /\p{N}/u.test(char);
      const isAllowedSpecial = ALLOWED_SPECIAL_CHARS.has(char);

      if (!isLetter && !isNumber && !isAllowedSpecial) {
        return {
          valid: false,
          error: t("validation.invalid_character", { char }),
        };
      }
    }
  }

  return { valid: true };
}

/**
 * Gets a user-friendly description of allowed characters
 * @param t - Translation function from i18next
 */
export function getAllowedCharactersDescription(t: TFunction): string {
  return t("validation.allowed_chars_description");
}
