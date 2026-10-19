export const joinUrls = (...parts: string[]): string => {
  const leadingSlash = /^\/+/;
  const trailingSlash = /\/+$/;

  const hasLeadingSlash = parts[0]?.startsWith("/") ?? false;

  const cleanedParts = parts.map((part, i) => {
    const isLast = i === parts.length - 1;
    const cleaned = part.replace(leadingSlash, "");
    return isLast ? cleaned : cleaned.replace(trailingSlash, "");
  });

  const result = cleanedParts.filter(Boolean).join("/");
  return hasLeadingSlash ? "/" + result : result;
};

export const formatUrl = (
  protocol?: string,
  hostname?: string,
  port?: string,
): string => {
  if (!protocol || !hostname) return "";

  const normalizedProtocol = protocol.endsWith(":") ? protocol : `${protocol}:`;

  const isDefaultPort =
    (normalizedProtocol === "http:" && port === "80") ||
    (normalizedProtocol === "https:" && port === "443");

  const portSegment = port && !isDefaultPort ? `:${port}` : "";

  return `${normalizedProtocol}//${hostname}${portSegment}`;
};

export const extractParts = (url: URL) => {
  const { protocol, hostname, port } = url;
  const isDefaultPort =
    (protocol === "http:" && port === "80") ||
    (protocol === "https:" && port === "443");
  const portPart = port && !isDefaultPort ? `:${port}` : "";
  return { protocol, hostname, portPart };
};

// Keep the target hostname while advertising the configured external scheme/port.
export const formatProxyUrl = (
  protocol: string,
  hostname: string,
  port?: string,
  publicUrl?: string,
): string => {
  if (publicUrl) {
    const origin = new URL(publicUrl);
    return formatUrl(origin.protocol, hostname, origin.port);
  }
  return formatUrl(protocol, hostname, port);
};
