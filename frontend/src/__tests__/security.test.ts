import { describe, it, expect } from 'vitest';
import {
  isAllowedInternalDownloadUrl,
  validatePreviewContent,
} from '../utils/security';

describe('Security: isAllowedInternalDownloadUrl', () => {
  const base = 'http://localhost:5173';

  it('allows same-origin internal download and proxy routes', () => {
    expect(isAllowedInternalDownloadUrl('/d/test.pdf', base)).toBe(true);
    expect(isAllowedInternalDownloadUrl('/p/image.png', base)).toBe(true);
    expect(isAllowedInternalDownloadUrl('/sd/share-id/file.pdf', base)).toBe(true);
    expect(isAllowedInternalDownloadUrl('/sp/share-id/file.png', base)).toBe(true);
    expect(isAllowedInternalDownloadUrl('http://localhost:5173/d/sub/doc.pdf', base)).toBe(true);
  });

  it('strictly rejects cross-origin URLs to prevent SSRF and token leakage', () => {
    expect(isAllowedInternalDownloadUrl('https://malicious.com/d/file.pdf', base)).toBe(false);
    expect(isAllowedInternalDownloadUrl('http://attacker.com/p/test', base)).toBe(false);
    expect(isAllowedInternalDownloadUrl('//attacker.com/d/file', base)).toBe(false);
  });

  it('strictly rejects non-download endpoints on the same origin', () => {
    expect(isAllowedInternalDownloadUrl('/api/me', base)).toBe(false);
    expect(isAllowedInternalDownloadUrl('/api/auth/login', base)).toBe(false);
    expect(isAllowedInternalDownloadUrl('/admin/user', base)).toBe(false);
    expect(isAllowedInternalDownloadUrl('/static/bundle.js', base)).toBe(false);
  });

  it('handles invalid or empty inputs safely', () => {
    expect(isAllowedInternalDownloadUrl('', base)).toBe(false);
    expect(isAllowedInternalDownloadUrl('invalid-url-string', base)).toBe(false);
  });
});

describe('Security: validatePreviewContent', () => {
  it('blocks HTML disguised with application/pdf MIME or PDF extension', () => {
    const htmlPayload = new TextEncoder().encode('<!doctype html><html><body><h1>Hacked</h1></body></html>');
    const result = validatePreviewContent('application/pdf', htmlPayload, 'document.pdf');
    expect(result.valid).toBe(false);
    expect(result.error).toContain('HTML');
  });

  it('blocks SVG disguised as PNG', () => {
    const svgPayload = new TextEncoder().encode('<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>');
    const result = validatePreviewContent('image/png', svgPayload, 'image.png');
    expect(result.valid).toBe(false);
  });

  it('blocks dangerous MIME types regardless of file name', () => {
    const dummyBytes = new Uint8Array([0x00, 0x01, 0x02, 0x03]);
    expect(validatePreviewContent('text/html', dummyBytes, 'doc.pdf').valid).toBe(false);
    expect(validatePreviewContent('image/svg+xml', dummyBytes, 'pic.png').valid).toBe(false);
    expect(validatePreviewContent('application/xhtml+xml', dummyBytes, 'test.jpg').valid).toBe(false);
    expect(validatePreviewContent('application/javascript', dummyBytes, 'code.png').valid).toBe(false);
  });

  it('validates authentic PDF files with %PDF- signature', () => {
    const pdfBytes = new TextEncoder().encode('%PDF-1.7\n1 0 obj\n...');
    const result = validatePreviewContent('application/pdf', pdfBytes, 'report.pdf');
    expect(result.valid).toBe(true);
    expect(result.detectedType).toBe('pdf');
    expect(result.mimeType).toBe('application/pdf');
  });

  it('rejects files claiming to be PDF without %PDF- magic bytes', () => {
    const fakePdfBytes = new Uint8Array([0x00, 0x11, 0x22, 0x33, 0x44]);
    const result = validatePreviewContent('application/pdf', fakePdfBytes, 'fake.pdf');
    expect(result.valid).toBe(false);
    expect(result.error).toContain('PDF imzası');
  });

  it('validates authentic PNG files with 89 50 4E 47 signature', () => {
    const pngBytes = new Uint8Array([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
    const result = validatePreviewContent('image/png', pngBytes, 'photo.png');
    expect(result.valid).toBe(true);
    expect(result.detectedType).toBe('raster');
    expect(result.mimeType).toBe('image/png');
  });

  it('validates authentic JPEG files with FF D8 FF signature', () => {
    const jpegBytes = new Uint8Array([0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10]);
    const result = validatePreviewContent('image/jpeg', jpegBytes, 'photo.jpg');
    expect(result.valid).toBe(true);
    expect(result.detectedType).toBe('raster');
    expect(result.mimeType).toBe('image/jpeg');
  });

  it('validates authentic WebP files with RIFF....WEBP signature', () => {
    const webpBytes = new Uint8Array([
      0x52, 0x49, 0x46, 0x46, // RIFF
      0x00, 0x00, 0x00, 0x00, // length
      0x57, 0x45, 0x42, 0x50, // WEBP
      0x56, 0x50, 0x38, 0x20, // VP8
    ]);
    const result = validatePreviewContent('image/webp', webpBytes, 'photo.webp');
    expect(result.valid).toBe(true);
    expect(result.detectedType).toBe('raster');
    expect(result.mimeType).toBe('image/webp');
  });
});
