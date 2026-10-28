export function formatBytes(bytes: number, decimals = 1): string {
  if (!bytes || bytes <= 0) return '0 B';
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const safeI = Math.min(i, sizes.length - 1);
  return `${parseFloat((bytes / Math.pow(k, safeI)).toFixed(dm))} ${sizes[safeI]}`;
}

export function formatDate(dateInput: string | Date | undefined | null): string {
  if (!dateInput) return '-';
  try {
    const d = typeof dateInput === 'string' ? new Date(dateInput) : dateInput;
    if (isNaN(d.getTime())) return '-';
    return d.toLocaleString('tr-TR', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return '-';
  }
}

export function joinPath(...parts: string[]): string {
  const cleaned: string[] = [];
  for (const part of parts) {
    if (!part) continue;
    const segments = part.replace(/\\/g, '/').split('/');
    for (const segment of segments) {
      if (!segment || segment === '.') continue;
      if (segment === '..') {
        cleaned.pop();
      } else {
        cleaned.push(segment);
      }
    }
  }
  return '/' + cleaned.join('/');
}

export function splitPath(fullPath: string): { dir: string; name: string } {
  const normalized = joinPath(fullPath);
  if (normalized === '/') {
    return { dir: '/', name: '' };
  }
  const lastSlashIndex = normalized.lastIndexOf('/');
  const dir = lastSlashIndex <= 0 ? '/' : normalized.substring(0, lastSlashIndex);
  const name = normalized.substring(lastSlashIndex + 1);
  return { dir, name };
}

export function getBreadcrumbs(currentPath: string): Array<{ name: string; path: string }> {
  const normalized = joinPath(currentPath);
  const breadcrumbs: Array<{ name: string; path: string }> = [
    { name: 'Kök Dizin', path: '/' },
  ];
  if (normalized === '/') return breadcrumbs;

  const segments = normalized.split('/').filter(Boolean);
  let accumulated = '';
  for (const segment of segments) {
    accumulated += '/' + segment;
    breadcrumbs.push({
      name: segment,
      path: accumulated,
    });
  }
  return breadcrumbs;
}

export function getFileExtension(name: string): string {
  const dotIndex = name.lastIndexOf('.');
  if (dotIndex <= 0 || dotIndex === name.length - 1) return '';
  return name.substring(dotIndex + 1).toLowerCase();
}

/**
 * Only inline raster images (JPEG, PNG, GIF, WebP).
 * SVG is explicitly excluded for security (download-only).
 */
export function isInlineRasterImage(name: string): boolean {
  const ext = getFileExtension(name);
  return ['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext);
}

export function isPdf(name: string): boolean {
  return getFileExtension(name) === 'pdf';
}

export function isSvgOrHtml(name: string): boolean {
  const ext = getFileExtension(name);
  return ['svg', 'html', 'htm'].includes(ext);
}

export const MAX_PREVIEW_SIZE_BYTES = 20 * 1024 * 1024; // 20 MiB

export function getPreviewEligibility(
  name: string,
  size?: number
): { canPreview: boolean; reason?: string; type: 'raster' | 'pdf' | 'none' } {
  if (isSvgOrHtml(name)) {
    return {
      canPreview: false,
      reason: 'Güvenlik nedeniyle SVG ve HTML dosyaları önizlenmez, doğrudan indirilir.',
      type: 'none',
    };
  }

  const isRaster = isInlineRasterImage(name);
  const isPdfFile = isPdf(name);

  if (!isRaster && !isPdfFile) {
    return {
      canPreview: false,
      reason: 'Bu dosya türü için tarayıcı içi önizleme desteklenmiyor.',
      type: 'none',
    };
  }

  if (size !== undefined && size > MAX_PREVIEW_SIZE_BYTES) {
    return {
      canPreview: false,
      reason: `Dosya boyutu 20 MB sınırını aşıyor (${formatBytes(size)}). Lütfen dosyayı indirin.`,
      type: isRaster ? 'raster' : 'pdf',
    };
  }

  return {
    canPreview: true,
    type: isRaster ? 'raster' : 'pdf',
  };
}

export const ALLOWED_TEXT_EXTENSIONS = [
  'txt',
  'md',
  'csv',
  'json',
  'log',
  'yaml',
  'yml',
  'xml',
  'ini',
  'conf',
  'toml',
  'css',
  'js',
  'ts',
  'html',
  'svg',
] as const;

export function isEditableTextFile(name: string): boolean {
  const ext = getFileExtension(name);
  return (ALLOWED_TEXT_EXTENSIONS as readonly string[]).includes(ext);
}

export const MAX_TEXT_SIZE_BYTES = 2 * 1024 * 1024; // 2 MiB

/**
 * Calculates exact byte length in UTF-8 encoding (not JS UTF-16 code units length).
 */
export function getUtf8ByteLength(text: string): number {
  return new TextEncoder().encode(text).length;
}

export function canEditTextFile(
  name: string,
  size?: number
): { canEdit: boolean; reason?: string } {
  if (!isEditableTextFile(name)) {
    return {
      canEdit: false,
      reason: 'Bu dosya uzantısı metin düzenleyici tarafından desteklenmiyor.',
    };
  }

  if (size !== undefined && size > MAX_TEXT_SIZE_BYTES) {
    return {
      canEdit: false,
      reason: `Dosya boyutu 2 MB metin düzenleme sınırını aşıyor (${formatBytes(size)}).`,
    };
  }

  return { canEdit: true };
}

