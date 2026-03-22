const ACCESS_TOKEN_COOKIE = "forklore_access_token";
const REFRESH_TOKEN_COOKIE = "forklore_refresh_token";

function isBrowser(): boolean {
  return typeof document !== "undefined";
}

function buildCookie(name: string, value: string, maxAgeSeconds: number): string {
  const secure = typeof window !== "undefined" && window.location.protocol === "https:" ? "; Secure" : "";
  return `${name}=${encodeURIComponent(value)}; Path=/; Max-Age=${maxAgeSeconds}; SameSite=Lax${secure}`;
}

function readCookie(name: string): string | null {
  if (!isBrowser()) {
    return null;
  }

  const encodedName = `${name}=`;
  const cookies = document.cookie ? document.cookie.split(";") : [];

  for (const part of cookies) {
    const cookie = part.trim();
    if (cookie.startsWith(encodedName)) {
      return decodeURIComponent(cookie.slice(encodedName.length));
    }
  }

  return null;
}

function deleteCookie(name: string): void {
  if (!isBrowser()) {
    return;
  }

  document.cookie = `${name}=; Path=/; Max-Age=0; SameSite=Lax`;
}

export function setAuthTokens(accessToken: string, refreshToken: string): void {
  if (!isBrowser()) {
    return;
  }

  document.cookie = buildCookie(ACCESS_TOKEN_COOKIE, accessToken, 60 * 60 * 24 * 7);
  document.cookie = buildCookie(REFRESH_TOKEN_COOKIE, refreshToken, 60 * 60 * 24 * 30);
}

export function getAccessToken(): string | null {
  return readCookie(ACCESS_TOKEN_COOKIE);
}

export function getRefreshToken(): string | null {
  return readCookie(REFRESH_TOKEN_COOKIE);
}

export function clearAuthTokens(): void {
  deleteCookie(ACCESS_TOKEN_COOKIE);
  deleteCookie(REFRESH_TOKEN_COOKIE);
}
