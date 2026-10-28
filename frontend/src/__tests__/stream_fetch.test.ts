import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { fsApi, authApi, ApiError, setStoredToken } from '../api/client';

describe('Actual stream reader & security tests', () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    setStoredToken('test-admin-token');
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
    setStoredToken(null);
  });

  function createReadableStream(chunks: Uint8Array[], onCancel?: () => void): ReadableStream<Uint8Array> {
    let index = 0;
    return new ReadableStream<Uint8Array>({
      pull(controller) {
        if (index < chunks.length) {
          controller.enqueue(chunks[index]);
          index++;
        } else {
          controller.close();
        }
      },
      cancel() {
        if (onCancel) onCancel();
      },
    });
  }

  it('successfully validates split PDF signature delivered in 1-byte network chunks', async () => {
    // "%PDF-1.4..." split into 1-byte chunks
    const fullHeader = '%PDF-1.4 valid pdf content here';
    const chunks: Uint8Array[] = Array.from(fullHeader).map((ch) => new TextEncoder().encode(ch));

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      redirected: false,
      headers: new Headers({
        'content-type': 'application/pdf',
      }),
      body: createReadableStream(chunks),
    } as unknown as Response);

    const blob = await fsApi.fetchAuthorizedBlob('/d/test.pdf', 'test.pdf', 1024 * 1024);
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.type).toBe('application/pdf');
    expect(blob.size).toBe(chunks.length);
  });

  it('cancels stream reader and throws 413 when cap is exceeded with lying Content-Length header', async () => {
    // Server claims Content-Length is only 100 bytes, but streams 1.5 MB
    const largeChunk = new Uint8Array(500 * 1024); // 500 KB
    const chunks = [largeChunk, largeChunk, largeChunk, largeChunk, largeChunk];

    let cancelCalled = false;
    const stream = createReadableStream(chunks, () => {
      cancelCalled = true;
    });

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      redirected: false,
      headers: new Headers({
        'content-type': 'application/pdf',
        'content-length': '100', // lying Content-Length header!
      }),
      body: stream,
    } as unknown as Response);

    // Limit cap to 1 MB (1024 * 1024 bytes)
    await expect(
      fsApi.fetchAuthorizedBlob('/d/test.pdf', 'test.pdf', 1024 * 1024)
    ).rejects.toThrowError(/sınırını aşıyor/);

    expect(cancelCalled).toBe(true);
  });

  it('cancels stream reader and throws 413 when cap is exceeded with absent Content-Length header', async () => {
    const chunk1 = new Uint8Array(600 * 1024);
    const chunk2 = new Uint8Array(600 * 1024);
    let cancelCalled = false;

    const stream = createReadableStream([chunk1, chunk2], () => {
      cancelCalled = true;
    });

    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      redirected: false,
      headers: new Headers({
        'content-type': 'application/pdf',
        // No content-length header!
      }),
      body: stream,
    } as unknown as Response);

    await expect(
      fsApi.fetchAuthorizedBlob('/d/test.pdf', 'test.pdf', 500 * 1024)
    ).rejects.toThrowError(/sınırını aşıyor/);

    expect(cancelCalled).toBe(true);
  });

  it('cancels and rejects with error when AbortSignal triggers', async () => {
    const controller = new AbortController();

    globalThis.fetch = vi.fn().mockImplementation((_url, opts) => {
      return new Promise((_, reject) => {
        if (opts?.signal) {
          opts.signal.addEventListener('abort', () => {
            reject(new DOMException('The user aborted a request.', 'AbortError'));
          });
        }
      });
    });

    const promise = fsApi.fetchAuthorizedBlob('/d/test.pdf', 'test.pdf', 1024 * 1024, controller.signal);
    controller.abort();

    await expect(promise).rejects.toThrowError(/iptal edildi/);
  });

  it('rejects external/cross-origin URL before fetch without leaking Authorization token', async () => {
    const fetchSpy = vi.fn();
    globalThis.fetch = fetchSpy;

    await expect(
      fsApi.fetchAuthorizedBlob('https://attacker.com/steal-token', 'test.pdf')
    ).rejects.toThrowError(/Güvenlik uyarısı/);

    // Fetch was never called!
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it('rejects HTTP failure status even if response envelope body contains code: 200', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
      headers: new Headers({ 'content-type': 'application/json' }),
      json: () => Promise.resolve({ code: 200, message: 'Spoofed success in 500 body', data: {} }),
    } as unknown as Response);

    await expect(authApi.me()).rejects.toThrowError(ApiError);
    await expect(authApi.me()).rejects.toMatchObject({
      status: 500,
    });
  });
});
