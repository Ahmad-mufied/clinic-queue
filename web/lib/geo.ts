import type { AuditLog } from "./types";

/**
 * Default physical premise location for the clinic.
 */
export const DEFAULT_CLINIC_LOCATION = "Yogyakarta, Indonesia";

/**
 * Checks if an IP string is private, loopback, link-local, or local development network.
 */
export function isPrivateOrLocalIP(ip?: string | null): boolean {
  if (!ip) return true;
  let trimmed = ip.trim();
  if (!trimmed) return true;

  // Handle host:port (e.g. 127.0.0.1:8080 or [::1]:8080)
  if (trimmed.startsWith("[") && trimmed.includes("]")) {
    trimmed = trimmed.substring(1, trimmed.indexOf("]"));
  } else if (trimmed.includes(":") && !trimmed.includes("::")) {
    // IPv4 with port (e.g. 127.0.0.1:8080)
    const colonIdx = trimmed.indexOf(":");
    trimmed = trimmed.substring(0, colonIdx);
  }

  // Loopback
  if (
    trimmed === "127.0.0.1" ||
    trimmed === "::1" ||
    trimmed.startsWith("127.") ||
    trimmed.toLowerCase() === "localhost"
  ) {
    return true;
  }

  // IPv4-mapped IPv6 loopback / private (e.g. ::ffff:127.0.0.1)
  if (
    trimmed.startsWith("::ffff:127.") ||
    trimmed.startsWith("::ffff:192.168.") ||
    trimmed.startsWith("::ffff:10.")
  ) {
    return true;
  }

  // RFC 1918 Private Ranges
  if (trimmed.startsWith("10.")) return true;
  if (trimmed.startsWith("192.168.")) return true;
  if (trimmed.startsWith("169.254.")) return true; // Link-local
  if (trimmed.startsWith("fc") || trimmed.startsWith("fe80")) return true; // IPv6 local

  // 172.16.0.0 – 172.31.255.255
  const match172 = trimmed.match(/^172\.(\d+)\./);
  if (match172) {
    const octet = parseInt(match172[1], 10);
    if (octet >= 16 && octet <= 31) return true;
  }

  return false;
}

/**
 * Formats a human-readable location string for an audit log entry.
 * Prioritizes pre-enriched location, falls back to IP-based resolution.
 */
export function formatAuditLocation(
  log: Pick<AuditLog, "location" | "ip_address" | "details">
): string {
  if (log.location && log.location.trim() !== "") {
    return log.location.trim();
  }
  if (log.details?.location && typeof log.details.location === "string" && log.details.location.trim() !== "") {
    return log.details.location.trim();
  }

  if (isPrivateOrLocalIP(log.ip_address)) {
    return DEFAULT_CLINIC_LOCATION;
  }

  return "Indonesia";
}
