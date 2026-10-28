import { describe, it, expect } from 'vitest';
import {
  isInlineRasterImage,
  isPdf,
  isSvgOrHtml,
  getPreviewEligibility,
  MAX_PREVIEW_SIZE_BYTES,
} from '../utils/format';

describe('Preview format and security rules', () => {
  it('identifies inline raster image formats correctly', () => {
    expect(isInlineRasterImage('photo.jpg')).toBe(true);
    expect(isInlineRasterImage('PHOTO.JPEG')).toBe(true);
    expect(isInlineRasterImage('graphic.png')).toBe(true);
    expect(isInlineRasterImage('animation.gif')).toBe(true);
    expect(isInlineRasterImage('modern.webp')).toBe(true);
    // Non-raster or active formats must be false
    expect(isInlineRasterImage('vector.svg')).toBe(false);
    expect(isInlineRasterImage('doc.pdf')).toBe(false);
    expect(isInlineRasterImage('page.html')).toBe(false);
  });

  it('identifies PDF format correctly', () => {
    expect(isPdf('document.pdf')).toBe(true);
    expect(isPdf('REPORT.PDF')).toBe(true);
    expect(isPdf('photo.jpg')).toBe(false);
  });

  it('identifies active formats SVG and HTML for download-only enforcement', () => {
    expect(isSvgOrHtml('icon.svg')).toBe(true);
    expect(isSvgOrHtml('index.html')).toBe(true);
    expect(isSvgOrHtml('page.htm')).toBe(true);
    expect(isSvgOrHtml('image.png')).toBe(false);
  });

  describe('getPreviewEligibility', () => {
    it('strictly forbids SVG and HTML from inline preview', () => {
      const svgResult = getPreviewEligibility('malicious.svg', 1024);
      expect(svgResult.canPreview).toBe(false);
      expect(svgResult.type).toBe('none');
      expect(svgResult.reason).toContain('SVG ve HTML dosyaları önizlenmez');

      const htmlResult = getPreviewEligibility('page.html', 2048);
      expect(htmlResult.canPreview).toBe(false);
      expect(htmlResult.type).toBe('none');
    });

    it('allows valid raster images under 20 MiB', () => {
      const res = getPreviewEligibility('photo.png', 5 * 1024 * 1024);
      expect(res.canPreview).toBe(true);
      expect(res.type).toBe('raster');
    });

    it('allows valid PDF documents under 20 MiB', () => {
      const res = getPreviewEligibility('report.pdf', 10 * 1024 * 1024);
      expect(res.canPreview).toBe(true);
      expect(res.type).toBe('pdf');
    });

    it('caps preview size at 20 MiB and rejects oversized files', () => {
      const oversizedBytes = MAX_PREVIEW_SIZE_BYTES + 1;
      const res = getPreviewEligibility('huge_image.jpg', oversizedBytes);
      expect(res.canPreview).toBe(false);
      expect(res.reason).toContain('20 MB sınırını aşıyor');
    });

    it('rejects unsupported file formats gracefully', () => {
      const res = getPreviewEligibility('archive.zip', 1024);
      expect(res.canPreview).toBe(false);
      expect(res.type).toBe('none');
      expect(res.reason).toContain('desteklenmiyor');
    });
  });
});
