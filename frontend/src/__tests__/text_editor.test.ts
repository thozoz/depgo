import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  isEditableTextFile,
  canEditTextFile,
  getUtf8ByteLength,
  MAX_TEXT_SIZE_BYTES,
  ALLOWED_TEXT_EXTENSIONS,
} from '../utils/format';
import { textApi, setStoredToken } from '../api/client';

describe('Text editor utilities and API contract', () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    setStoredToken('test-admin-token');
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
    setStoredToken(null);
  });

  describe('Extension allowlist', () => {
    it('accepts all 16 allowlisted UTF-8 text extensions', () => {
      const expected = [
        'txt', 'md', 'csv', 'json', 'log', 'yaml', 'yml', 'xml',
        'ini', 'conf', 'toml', 'css', 'js', 'ts', 'html', 'svg'
      ];
      expect(ALLOWED_TEXT_EXTENSIONS).toEqual(expected);

      for (const ext of expected) {
        expect(isEditableTextFile(`file.${ext}`)).toBe(true);
        expect(isEditableTextFile(`file.${ext.toUpperCase()}`)).toBe(true);
      }
    });

    it('rejects binary and un-allowlisted extensions', () => {
      expect(isEditableTextFile('binary.exe')).toBe(false);
      expect(isEditableTextFile('archive.zip')).toBe(false);
      expect(isEditableTextFile('photo.png')).toBe(false);
      expect(isEditableTextFile('document.pdf')).toBe(false);
    });
  });

  describe('UTF-8 byte length calculation vs JS UTF-16 length', () => {
    it('calculates ASCII byte length matching JS length', () => {
      const ascii = 'Hello world';
      expect(getUtf8ByteLength(ascii)).toBe(11);
      expect(ascii.length).toBe(11);
    });

    it('correctly calculates multi-byte Turkish characters as 2 bytes each', () => {
      const turkishText = 'şğüöçİ'; // 6 Turkish characters
      expect(turkishText.length).toBe(6); // UTF-16 code units
      expect(getUtf8ByteLength(turkishText)).toBe(12); // UTF-8 bytes: 2 bytes each
    });

    it('correctly calculates 4-byte emoji sequences', () => {
      const emojiText = '🚀🔥'; // 2 emojis
      expect(emojiText.length).toBe(4); // surrogate pairs in UTF-16
      expect(getUtf8ByteLength(emojiText)).toBe(8); // 4 bytes each in UTF-8
    });
  });

  describe('canEditTextFile size capping', () => {
    it('approves editable file under 2 MiB', () => {
      const res = canEditTextFile('notes.md', 100 * 1024);
      expect(res.canEdit).toBe(true);
    });

    it('rejects file exceeding 2 MiB limit', () => {
      const res = canEditTextFile('notes.md', MAX_TEXT_SIZE_BYTES + 1);
      expect(res.canEdit).toBe(false);
      expect(res.reason).toContain('2 MB');
    });

    it('rejects non-allowlisted file format', () => {
      const res = canEditTextFile('app.apk', 500);
      expect(res.canEdit).toBe(false);
      expect(res.reason).toContain('desteklenmiyor');
    });
  });

  describe('textApi HTTP and revision conflict handling', () => {
    it('fetches text content and revision cleanly on GET /api/fs/text', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: () => Promise.resolve({
          code: 200,
          message: 'success',
          data: { content: '# Hello Depgo', revision: 4 },
        }),
      } as unknown as Response);

      const res = await textApi.get('/docs/hello.md');
      expect(res.content).toBe('# Hello Depgo');
      expect(res.revision).toBe(4);
    });

    it('propagates 409/412 revision conflict on PUT /api/fs/text without auto-retrying', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        statusText: 'Conflict',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: () => Promise.resolve({
          code: 409,
          message: 'revision mismatch',
        }),
      } as unknown as Response);

      await expect(textApi.put('/docs/hello.md', 'new content', 3)).rejects.toMatchObject({
        status: 409,
        code: 409,
        message: 'revision mismatch',
      });
    });
  });
});
