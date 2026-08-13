export interface TenantSummary {
  id: string;
  slug: string;
  name: string;
  timezone: string;
}

export interface CurrentUser {
  id: string;
  tenant: TenantSummary;
  email: string;
  fullName: string;
  roles: string[];
  permissions: string[];
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  tokenType: "Bearer";
  expiresIn: number;
}

export interface ResponseMeta {
  requestId?: string;
}

export interface AuthEnvelope {
  data: {
    tokens: TokenPair;
    user: CurrentUser;
  };
  meta?: ResponseMeta;
}

export interface TokenEnvelope {
  data: TokenPair;
  meta?: ResponseMeta;
}

export interface MeEnvelope {
  data: CurrentUser;
  meta?: ResponseMeta;
}

export interface LogoutEnvelope {
  data: {
    loggedOut: true;
  };
  meta?: ResponseMeta;
}

export interface ErrorDetail {
  field?: string;
  code: string;
  message: string;
}

export interface ErrorEnvelope {
  error: {
    code: string;
    message: string;
    requestId: string;
    details?: ErrorDetail[];
  };
}

type JsonObject = Record<string, unknown>;

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
const TENANT_SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
const ROLE_PATTERN = /^[a-z][a-z0-9_]{1,62}$/;
const PERMISSION_PATTERN = /^[a-z][a-z0-9_]*:[a-z][a-z0-9_]*$/;
const ERROR_CODE_PATTERN = /^[a-z][a-z0-9_]*$/;

function isObject(value: unknown): value is JsonObject {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function hasExactKeys(value: JsonObject, required: string[], optional: string[] = []): boolean {
  const allowed = new Set([...required, ...optional]);
  return required.every((key) => Object.hasOwn(value, key)) && Object.keys(value).every((key) => allowed.has(key));
}

function isUniqueStringArray(value: unknown, pattern: RegExp): value is string[] {
  return (
    Array.isArray(value) &&
    value.every((item) => typeof item === "string" && pattern.test(item)) &&
    new Set(value).size === value.length
  );
}

function isMeta(value: unknown): value is ResponseMeta {
  return (
    isObject(value) &&
    hasExactKeys(value, [], ["requestId"]) &&
    (!Object.hasOwn(value, "requestId") || typeof value.requestId === "string")
  );
}

function hasValidOptionalMeta(value: JsonObject): boolean {
  return !Object.hasOwn(value, "meta") || isMeta(value.meta);
}

function isTenant(value: unknown): value is TenantSummary {
  return (
    isObject(value) &&
    hasExactKeys(value, ["id", "slug", "name", "timezone"]) &&
    typeof value.id === "string" &&
    UUID_PATTERN.test(value.id) &&
    typeof value.slug === "string" &&
    value.slug.length >= 1 &&
    value.slug.length <= 63 &&
    TENANT_SLUG_PATTERN.test(value.slug) &&
    typeof value.name === "string" &&
    typeof value.timezone === "string"
  );
}

export function isCurrentUser(value: unknown): value is CurrentUser {
  return (
    isObject(value) &&
    hasExactKeys(value, ["id", "tenant", "email", "fullName", "roles", "permissions"]) &&
    typeof value.id === "string" &&
    UUID_PATTERN.test(value.id) &&
    isTenant(value.tenant) &&
    typeof value.email === "string" &&
    value.email.length <= 320 &&
    value.email.includes("@") &&
    typeof value.fullName === "string" &&
    isUniqueStringArray(value.roles, ROLE_PATTERN) &&
    isUniqueStringArray(value.permissions, PERMISSION_PATTERN)
  );
}

function isTokenPair(value: unknown): value is TokenPair {
  return (
    isObject(value) &&
    hasExactKeys(value, ["accessToken", "refreshToken", "tokenType", "expiresIn"]) &&
    typeof value.accessToken === "string" &&
    value.accessToken.length >= 1 &&
    typeof value.refreshToken === "string" &&
    value.refreshToken.length >= 32 &&
    value.refreshToken.length <= 2048 &&
    value.tokenType === "Bearer" &&
    typeof value.expiresIn === "number" &&
    Number.isInteger(value.expiresIn) &&
    value.expiresIn >= 1
  );
}

export function parseAuthEnvelope(value: unknown): AuthEnvelope | null {
  if (!isObject(value) || !hasExactKeys(value, ["data"], ["meta"]) || !hasValidOptionalMeta(value)) {
    return null;
  }
  if (!isObject(value.data) || !hasExactKeys(value.data, ["tokens", "user"])) {
    return null;
  }
  if (!isTokenPair(value.data.tokens) || !isCurrentUser(value.data.user)) {
    return null;
  }
  return value as unknown as AuthEnvelope;
}

export function parseTokenEnvelope(value: unknown): TokenEnvelope | null {
  if (!isObject(value) || !hasExactKeys(value, ["data"], ["meta"]) || !hasValidOptionalMeta(value)) {
    return null;
  }
  return isTokenPair(value.data) ? (value as unknown as TokenEnvelope) : null;
}

export function parseMeEnvelope(value: unknown): MeEnvelope | null {
  if (!isObject(value) || !hasExactKeys(value, ["data"], ["meta"]) || !hasValidOptionalMeta(value)) {
    return null;
  }
  return isCurrentUser(value.data) ? (value as unknown as MeEnvelope) : null;
}

export function parseLogoutEnvelope(value: unknown): LogoutEnvelope | null {
  if (!isObject(value) || !hasExactKeys(value, ["data"], ["meta"]) || !hasValidOptionalMeta(value)) {
    return null;
  }
  if (!isObject(value.data) || !hasExactKeys(value.data, ["loggedOut"]) || value.data.loggedOut !== true) {
    return null;
  }
  return value as unknown as LogoutEnvelope;
}

function isErrorDetail(value: unknown): value is ErrorDetail {
  return (
    isObject(value) &&
    hasExactKeys(value, ["code", "message"], ["field"]) &&
    typeof value.code === "string" &&
    typeof value.message === "string" &&
    (!Object.hasOwn(value, "field") || typeof value.field === "string")
  );
}

export function parseErrorEnvelope(value: unknown): ErrorEnvelope | null {
  if (!isObject(value) || !hasExactKeys(value, ["error"]) || !isObject(value.error)) {
    return null;
  }
  const error = value.error;
  if (!hasExactKeys(error, ["code", "message", "requestId"], ["details"])) {
    return null;
  }
  if (
    typeof error.code !== "string" ||
    !ERROR_CODE_PATTERN.test(error.code) ||
    typeof error.message !== "string" ||
    typeof error.requestId !== "string"
  ) {
    return null;
  }
  if (Object.hasOwn(error, "details") && (!Array.isArray(error.details) || !error.details.every(isErrorDetail))) {
    return null;
  }
  return value as unknown as ErrorEnvelope;
}
