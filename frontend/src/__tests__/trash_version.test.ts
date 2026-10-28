import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { trashApi, versionApi, setStoredToken, ApiError } from '../api/client';

describe('Trash and Version history API contracts', () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    setStoredToken('test-admin-token');
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
    setStoredToken(null);
  });

  describe('Trash API', () => {
    it('calls GET /api/fs/trash with query parameters and parses envelope', async () => {
      globalThis.fetch = vi.fn().mockImplementation((url) => {
        expect(url).toContain('/api/fs/trash?path=%2F&page=1&per_page=50');
        return Promise.resolve({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: () =>
            Promise.resolve({
              code: 200,
              message: 'success',
              data: {
                content: [
                  {
                    trash_id: 't-123',
                    original_path: '/old/path.txt',
                    size: 1024,
                    is_dir: false,
                    deleted_at: '2026-10-28T12:00:00Z',
                  },
                ],
                total: 1,
                page: 1,
                per_page: 50,
              },
            }),
        });
      });

      const res = await trashApi.list('/', 1, 50);
      expect(res.total).toBe(1);
      expect(res.content[0].trash_id).toBe('t-123');
      expect(res.content[0].original_path).toBe('/old/path.txt');
    });

    it('handles 409 collision on POST /api/fs/trash/restore', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        statusText: 'Conflict',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: () => Promise.resolve({ code: 409, message: 'file exists at destination' }),
      } as unknown as Response);

      await expect(trashApi.restore('t-123', '/')).rejects.toMatchObject({
        status: 409,
        code: 409,
      });
    });

    it('submits non-destructive target_path on POST /api/fs/trash/restore', async () => {
      let sentBody = '';
      globalThis.fetch = vi.fn().mockImplementation((_url, opts) => {
        sentBody = opts.body;
        return Promise.resolve({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: () => Promise.resolve({ code: 200, message: 'success', data: null }),
        });
      });

      await trashApi.restore('t-123', '/', '/new/custom/path.txt');
      const parsed = JSON.parse(sentBody);
      expect(parsed.trash_id).toBe('t-123');
      expect(parsed.target_path).toBe('/new/custom/path.txt');
    });

    it('sends explicit confirm: true on POST /api/fs/trash/delete', async () => {
      let sentBody = '';
      globalThis.fetch = vi.fn().mockImplementation((_url, opts) => {
        sentBody = opts.body;
        return Promise.resolve({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: () => Promise.resolve({ code: 200, message: 'success', data: null }),
        });
      });

      await trashApi.delete('t-999', '/');
      const parsed = JSON.parse(sentBody);
      expect(parsed.trash_id).toBe('t-999');
      expect(parsed.confirm).toBe(true);
    });
  });

  describe('Versions API', () => {
    it('calls GET /api/fs/versions and parses file_id and revision', async () => {
      globalThis.fetch = vi.fn().mockImplementation((url) => {
        expect(url).toContain('/api/fs/versions?path=%2Ffile.txt');
        return Promise.resolve({
          ok: true,
          status: 200,
          headers: new Headers({ 'content-type': 'application/json' }),
          json: () =>
            Promise.resolve({
              code: 200,
              message: 'success',
              data: {
                file_id: 'f-1',
                revision: 3,
                content: [
                  { version_id: 'v-2', revision: 2, size: 500, created_at: '2026-10-28T10:00:00Z' },
                  { version_id: 'v-1', revision: 1, size: 450, created_at: '2026-10-27T10:00:00Z' },
                ],
                total: 2,
                page: 1,
                per_page: 50,
              },
            }),
        });
      });

      const res = await versionApi.list('/file.txt', 1, 50);
      expect(res.revision).toBe(3);
      expect(res.content.length).toBe(2);
      expect(res.content[0].version_id).toBe('v-2');
    });

    it('rejects POST /api/fs/versions/restore on revision conflict (409/412)', async () => {
      globalThis.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 412,
        statusText: 'Precondition Failed',
        headers: new Headers({ 'content-type': 'application/json' }),
        json: () => Promise.resolve({ code: 412, message: 'revision mismatch' }),
      } as unknown as Response);

      await expect(versionApi.restore('/file.txt', 'v-1', 2)).rejects.toThrowError(ApiError);
      await expect(versionApi.restore('/file.txt', 'v-1', 2)).rejects.toMatchObject({
        status: 412,
        code: 412,
      });
    });
  });
});
