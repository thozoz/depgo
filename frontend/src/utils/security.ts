import { isInlineRasterImage, isPdf } from './format';

/**
 * Validates that a download/preview URL is strictly on the same origin
 * and targets an allowed internal download/proxy route (/d/, /p/, /sd/, /sp/).
 * Prevents SSRF / token leakage to external domains or unexpected routes.
 */
export function isAllowedInternalDownloadUrl(
  rawUrl: string,
  baseOrigin = typeof window !== 'undefined' && window.location ? window.location.origin : 'http://localhost:5173'
): boolean {
  if (!rawUrl || typeof rawUrl !== 'string') return false;

  try {
    const parsed = new URL(rawUrl, baseOrigin);

    // Strict origin match
    if (parsed.origin !== baseOrigin) {
      return false;
    }

    // Must be one of the known internal download or proxy paths
    const allowedPrefixes = ['/d/', '/p/', '/sd/', '/sp/'];
    const pathname = parsed.pathname;

    return allowedPrefixes.some((prefix) => pathname.startsWith(prefix));
  } catch {
    return false;
  }
}

export interface ContentValidationResult {
  valid: boolean;
  detectedType: 'raster' | 'pdf' | 'none';
  mimeType: string;
  error?: string;
}

/**
 * Validates content-type header and binary magic bytes to prevent
 * active content (HTML, SVG, XML, scripts) disguised as PDF or images.
 */
export function validatePreviewContent(
  contentTypeHeader: string | null,
  firstChunk: Uint8Array,
  fileName: string
): ContentValidationResult {
  const mime = (contentTypeHeader || '').toLowerCase().split(';')[0].trim();

  // Explicitly reject any active or dangerous MIME types
  const dangerousMimes = [
    'text/html',
    'application/xhtml+xml',
    'image/svg+xml',
    'text/xml',
    'application/xml',
    'text/javascript',
    'application/javascript',
  ];
  if (dangerousMimes.some((d) => mime.includes(d))) {
    return {
      valid: false,
      detectedType: 'none',
      mimeType: mime,
      error: 'Güvenlik nedeniyle HTML, SVG veya komut dosyası içerikleri önizlenemez.',
    };
  }

  // Sniff text beginning to catch HTML/SVG disguised with generic MIME (e.g. application/octet-stream)
  if (firstChunk.length >= 4) {
    try {
      const headerStr = new TextDecoder().decode(firstChunk.slice(0, 150)).trim().toLowerCase();
      if (
        headerStr.startsWith('<!doctype html') ||
        headerStr.startsWith('<html') ||
        headerStr.startsWith('<?xml') ||
        headerStr.startsWith('<svg') ||
        headerStr.includes('<script')
      ) {
        return {
          valid: false,
          detectedType: 'none',
          mimeType: mime,
          error: 'Dosya içeriği HTML/SVG olarak tespit edildi; güvenlik nedeniyle önizlenemez.',
        };
      }
    } catch {
      // Binary decode error is normal for genuine binary files
    }
  }

  // PDF verification
  if (isPdf(fileName)) {
    // Magic bytes for PDF: %PDF- (0x25, 0x50, 0x44, 0x46, 0x2D)
    if (
      firstChunk.length >= 5 &&
      firstChunk[0] === 0x25 &&
      firstChunk[1] === 0x50 &&
      firstChunk[2] === 0x44 &&
      firstChunk[3] === 0x46 &&
      firstChunk[4] === 0x2d
    ) {
      return {
        valid: true,
        detectedType: 'pdf',
        mimeType: 'application/pdf',
      };
    }
    return {
      valid: false,
      detectedType: 'none',
      mimeType: mime,
      error: 'Geçersiz PDF dosyası: Dosya başında geçerli bir PDF imzası (%PDF-) bulunamadı.',
    };
  }

  // Raster image verification
  if (isInlineRasterImage(fileName)) {
    // PNG: 89 50 4E 47
    const isPng =
      firstChunk.length >= 4 &&
      firstChunk[0] === 0x89 &&
      firstChunk[1] === 0x50 &&
      firstChunk[2] === 0x4e &&
      firstChunk[3] === 0x47;

    // JPEG: FF D8 FF
    const isJpeg =
      firstChunk.length >= 3 &&
      firstChunk[0] === 0xff &&
      firstChunk[1] === 0xd8 &&
      firstChunk[2] === 0xff;

    // GIF: 47 49 46 (GIF)
    const isGif =
      firstChunk.length >= 6 &&
      firstChunk[0] === 0x47 &&
      firstChunk[1] === 0x49 &&
      firstChunk[2] === 0x46;

    // WebP: RIFF....WEBP (52 49 46 46 .... 57 45 42 50)
    const isWebp =
      firstChunk.length >= 12 &&
      firstChunk[0] === 0x52 &&
      firstChunk[1] === 0x49 &&
      firstChunk[2] === 0x46 &&
      firstChunk[3] === 0x46 &&
      firstChunk[8] === 0x57 &&
      firstChunk[9] === 0x45 &&
      firstChunk[10] === 0x42 &&
      firstChunk[11] === 0x50;

    if (isPng) return { valid: true, detectedType: 'raster', mimeType: 'image/png' };
    if (isJpeg) return { valid: true, detectedType: 'raster', mimeType: 'image/jpeg' };
    if (isGif) return { valid: true, detectedType: 'raster', mimeType: 'image/gif' };
    if (isWebp) return { valid: true, detectedType: 'raster', mimeType: 'image/webp' };

    return {
      valid: false,
      detectedType: 'none',
      mimeType: mime,
      error: 'Geçersiz resim formatı: Desteklenen raster resim imzası (PNG, JPEG, GIF, WebP) bulunamadı.',
    };
  }

  return {
    valid: false,
    detectedType: 'none',
    mimeType: mime,
    error: 'Desteklenmeyen dosya formatı.',
  };
}
