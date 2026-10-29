import {
  Resp,
  User,
  LoginRespData,
  FsListResp,
  FsGetResp,
  ShareResp,
  ShareListResp,
  CreateShareReq,
  UpdateShareReq,
  PublicShareInfoResp,
  PublicShareListResp,
  PublicShareGetResp,
  TrashListResp,
  UsageResp,
  VersionsResp,
  TextGetResp,
  TextPutResp,
} from '../types';
import { isAllowedInternalDownloadUrl, validatePreviewContent } from '../utils/security';
import { t } from '../utils/i18n';

export class ApiError extends Error {
  status: number;
  code: number;

  constructor(status: number, code: number, message: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

// Token storage in memory + guarded sessionStorage
let memoryToken: string | null = null;
const TOKEN_KEY = 'depgo_auth_token';

function getStoredToken(): string | null {
  if (memoryToken) return memoryToken;
  try {
    const val = sessionStorage.getItem(TOKEN_KEY);
    if (val) {
      memoryToken = val;
      return val;
    }
  } catch {
    // Session storage unavailable (private window/security blocks)
  }
  return null;
}

export function setStoredToken(token: string | null): void {
  memoryToken = token;
  try {
    if (token) {
      sessionStorage.setItem(TOKEN_KEY, token);
    } else {
      sessionStorage.removeItem(TOKEN_KEY);
    }
  } catch {
    // Session storage blocked
  }
}

// Guest share token stored purely in memory, never logged
const shareTokenMap = new Map<string, string>();

export function getShareToken(shareId: string): string | undefined {
  return shareTokenMap.get(shareId);
}

export function setShareToken(shareId: string, token: string): void {
  shareTokenMap.set(shareId, token);
}

export function clearShareToken(shareId: string): void {
  shareTokenMap.delete(shareId);
}

let onAuthFailureCallback: (() => void) | null = null;

export function setOnAuthFailure(callback: (() => void) | null): void {
  onAuthFailureCallback = callback;
}

function handle401(): void {
  setStoredToken(null);
  if (onAuthFailureCallback) {
    onAuthFailureCallback();
  }
}

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const token = getStoredToken();
  const headers = new Headers(options.headers || {});

  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', token);
  }

  if (!headers.has('Content-Type') && options.body && typeof options.body === 'string') {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    handle401();
    throw new ApiError(401, 401, t('error.session_expired'));
  }

  // Reject HTTP failure regardless of envelope content
  if (!response.ok) {
    let errorMsg = t('error.server_error', { status: response.statusText || response.status });
    let errorCode = response.status;
    try {
      const errorJson = await response.json();
      if (errorJson && typeof errorJson === 'object') {
        if (errorJson.message) errorMsg = String(errorJson.message);
        if (typeof errorJson.code === 'number') errorCode = errorJson.code;
        if (errorJson.code === 401) handle401();
      }
    } catch {
      // not JSON, keep status text
    }
    throw new ApiError(response.status, errorCode, errorMsg);
  }

  let data: Resp<T>;
  try {
    data = await response.json();
  } catch {
    throw new ApiError(response.status, -1, t('error.invalid_json'));
  }

  if (data.code === 401) {
    handle401();
    throw new ApiError(response.status, data.code, data.message || t('error.session_invalid'));
  }

  if (data.code !== 200) {
    throw new ApiError(response.status, data.code, data.message || t('error.operation_failed'));
  }

  return data.data;
}

export const authApi = {
  async login(username: string, password: string): Promise<LoginRespData> {
    const data = await request<LoginRespData>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    setStoredToken(data.token);
    return data;
  },

  async me(): Promise<User> {
    return request<User>('/api/me', {
      method: 'GET',
    });
  },

  async logout(): Promise<void> {
    try {
      await request<void>('/api/auth/logout', {
        method: 'GET',
      });
    } finally {
      setStoredToken(null);
    }
  },

  async updateMe(body: { username?: string; password?: string; current_password?: string }): Promise<void> {
    await request<void>('/api/me/update', {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  isAuthenticated(): boolean {
    return !!getStoredToken();
  },
};

export const fsApi = {
  async list(path: string, password = '', refresh = false, page = 1, perPage = 100): Promise<FsListResp> {
    return request<FsListResp>('/api/fs/list', {
      method: 'POST',
      body: JSON.stringify({
        path,
        password,
        refresh,
        page,
        per_page: perPage,
      }),
    });
  },

  async get(path: string, password = ''): Promise<FsGetResp> {
    return request<FsGetResp>('/api/fs/get', {
      method: 'POST',
      body: JSON.stringify({
        path,
        password,
      }),
    });
  },

  async mkdir(path: string): Promise<void> {
    return request<void>('/api/fs/mkdir', {
      method: 'POST',
      body: JSON.stringify({ path }),
    });
  },

  async rename(path: string, name: string): Promise<void> {
    return request<void>('/api/fs/rename', {
      method: 'POST',
      body: JSON.stringify({
        path,
        name,
        overwrite: false,
      }),
    });
  },

  async move(srcDir: string, dstDir: string, names: string[]): Promise<void> {
    return request<void>('/api/fs/move', {
      method: 'POST',
      body: JSON.stringify({
        src_dir: srcDir,
        dst_dir: dstDir,
        names,
        overwrite: false,
      }),
    });
  },

  async remove(dir: string, names: string[]): Promise<void> {
    return request<void>('/api/fs/remove', {
      method: 'POST',
      body: JSON.stringify({
        dir,
        names,
      }),
    });
  },

  uploadStream(
    filePath: string,
    file: File,
    onProgress?: (percent: number, loaded: number, total: number) => void,
    signal?: AbortSignal,
    overwrite = false // Default FALSE - no silent overwrites!
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open('PUT', '/api/fs/put', true);

      const token = getStoredToken();
      if (token) {
        xhr.setRequestHeader('Authorization', token);
      }
      xhr.setRequestHeader('File-Path', encodeURIComponent(filePath));
      xhr.setRequestHeader('As-Task', 'false');
      xhr.setRequestHeader('Overwrite', overwrite ? 'true' : 'false');
      xhr.setRequestHeader('Last-Modified', String(file.lastModified || Date.now()));
      xhr.setRequestHeader('Content-Type', file.type || 'application/octet-stream');

      if (onProgress) {
        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable) {
            const percent = Math.round((event.loaded / event.total) * 100);
            onProgress(percent, event.loaded, event.total);
          }
        };
      }

      xhr.onload = () => {
        if (xhr.status === 401) {
          handle401();
          reject(new ApiError(401, 401, t('error.session_auth_error')));
          return;
        }

        try {
          const resp = JSON.parse(xhr.responseText) as Resp<unknown>;
          if (resp.code === 200) {
            resolve();
          } else {
            if (resp.code === 401) handle401();
            reject(new ApiError(xhr.status, resp.code, resp.message || t('error.upload_failed')));
          }
        } catch {
          if (xhr.status >= 200 && xhr.status < 300) {
            resolve();
          } else {
            reject(new ApiError(xhr.status, xhr.status, t('error.upload_error_status', { status: xhr.status })));
          }
        }
      };

      xhr.onerror = () => {
        reject(new ApiError(0, -1, t('error.network_upload_failed')));
      };

      if (signal) {
        signal.addEventListener('abort', () => {
          xhr.abort();
          reject(new ApiError(0, -2, t('error.upload_cancelled')));
        });
      }

      xhr.send(file);
    });
  },

  /**
   * Safely fetch file content as a bounded, validated Blob.
   *
   * Security constraints:
   * 1. Rejects non-same-origin URLs and non-internal routes (/d/, /p/, /sd/, /sp/).
   * 2. Sets redirect: 'error' to prevent header leakage.
   * 3. Streams response body with strict byte counter, aborting if maxBytes is exceeded (prevents lying/missing Content-Length DOS).
   * 4. Validates MIME type and magic bytes to reject active HTML/SVG/scripts.
   * 5. If isPublic=true, does NOT send admin Authorization header.
   */
  async fetchAuthorizedBlob(
    rawUrl: string,
    fileName: string,
    maxBytes = 20 * 1024 * 1024,
    signal?: AbortSignal,
    isPublic = false
  ): Promise<Blob> {
    if (!isAllowedInternalDownloadUrl(rawUrl)) {
      throw new ApiError(
        400,
        400,
        t('error.preview_security_origin')
      );
    }

    const headers = new Headers();
    if (!isPublic) {
      const token = getStoredToken();
      if (token) {
        headers.set('Authorization', token);
      }
    }

    // redirect: 'error' prevents header forwarding to unintended targets
    let response: Response;
    try {
      response = await fetch(rawUrl, {
        method: 'GET',
        headers,
        redirect: 'error',
        signal,
      });
    } catch (err: unknown) {
      if (signal?.aborted) {
        throw new ApiError(0, 0, t('error.preview_cancelled'));
      }
      throw new ApiError(
        400,
        400,
        err instanceof Error ? err.message : t('error.preview_rejected_network')
      );
    }

    if (response.status === 401) {
      if (!isPublic) handle401();
      throw new ApiError(401, 401, t('error.unauthorized'));
    }

    if (!response.ok) {
      throw new ApiError(response.status, response.status, t('error.file_fetch_failed', { status: response.status }));
    }

    if (response.redirected) {
      throw new ApiError(400, 400, t('error.preview_redirect_forbidden'));
    }

    // Stream reader with strict byte counter (does not trust Content-Length header alone)
    if (!response.body) {
      throw new ApiError(500, 500, t('error.response_body_unread'));
    }

    const reader = response.body.getReader();
    const chunks: Uint8Array[] = [];
    let receivedBytes = 0;

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        if (value) {
          receivedBytes += value.length;
          if (receivedBytes > maxBytes) {
            await reader.cancel();
            throw new ApiError(
              413,
              413,
              t('error.preview_size_exceeded', { limit: Math.round(maxBytes / (1024 * 1024)) })
            );
          }
          chunks.push(value);
        }
      }
    } catch (err: unknown) {
      if (err instanceof ApiError) throw err;
      if (signal?.aborted) throw new ApiError(0, 0, t('error.request_cancelled'));
      throw new ApiError(500, 500, t('error.stream_read_error'));
    }

    if (chunks.length === 0) {
      throw new ApiError(400, 400, t('error.empty_preview'));
    }

    // Accumulate up to the first 150 bytes across all received chunks for reliable signature validation
    const headerSliceLen = Math.min(receivedBytes, 150);
    const headerBuffer = new Uint8Array(headerSliceLen);
    let offset = 0;
    for (const chunk of chunks) {
      if (offset >= headerSliceLen) break;
      const toCopy = Math.min(chunk.length, headerSliceLen - offset);
      headerBuffer.set(chunk.subarray(0, toCopy), offset);
      offset += toCopy;
    }

    // Validate MIME and magic bytes on the accumulated header
    const validation = validatePreviewContent(
      response.headers.get('content-type'),
      headerBuffer,
      fileName
    );

    if (!validation.valid) {
      throw new ApiError(415, 415, validation.error || t('error.invalid_preview_format'));
    }

    return new Blob(chunks as BlobPart[], { type: validation.mimeType });
  },
};

export const shareApi = {
  async list(page = 1, perPage = 20): Promise<ShareListResp> {
    return request<ShareListResp>(`/api/share/list?page=${page}&per_page=${perPage}`, {
      method: 'GET',
    });
  },

  async create(data: CreateShareReq): Promise<ShareResp> {
    return request<ShareResp>('/api/share/create', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(data: UpdateShareReq): Promise<ShareResp> {
    return request<ShareResp>('/api/share/update', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async disable(shareId: string): Promise<void> {
    return request<void>('/api/share/disable', {
      method: 'POST',
      body: JSON.stringify({ share_id: shareId }),
    });
  },

  async enable(shareId: string): Promise<ShareResp> {
    return request<ShareResp>('/api/share/enable', {
      method: 'POST',
      body: JSON.stringify({ share_id: shareId }),
    });
  },

  async delete(shareId: string): Promise<void> {
    return request<void>('/api/share/delete', {
      method: 'POST',
      body: JSON.stringify({ share_id: shareId }),
    });
  },
};

export const publicShareApi = {
  async getInfo(shareId: string): Promise<PublicShareInfoResp> {
    const token = getShareToken(shareId);
    const headers: Record<string, string> = {};
    if (token) {
      // The header keeps the token out of the URL, and therefore out of browser
      // history, the Referer header and proxy access logs. Only media elements,
      // which cannot set headers, still fall back to the ?auth= form.
      headers['X-Share-Token'] = token;
    }
    return request<PublicShareInfoResp>(
      `/api/public/share/info?share_id=${encodeURIComponent(shareId)}`,
      {
        method: 'GET',
        headers,
      }
    );
  },

  async auth(shareId: string, password = ''): Promise<{ token: string }> {
    const data = await request<{ token: string }>('/api/public/share/auth', {
      method: 'POST',
      body: JSON.stringify({ share_id: shareId, password }),
    });
    if (data.token) {
      setShareToken(shareId, data.token);
    }
    return data;
  },

  async list(shareId: string, path = '/', page = 1, perPage = 100): Promise<PublicShareListResp> {
    const token = getShareToken(shareId) || '';
    return request<PublicShareListResp>('/api/public/share/list', {
      method: 'POST',
      body: JSON.stringify({
        share_id: shareId,
        path,
        token,
        page,
        per_page: perPage,
      }),
    });
  },

  async get(shareId: string, path = ''): Promise<PublicShareGetResp> {
    const token = getShareToken(shareId) || '';
    return request<PublicShareGetResp>('/api/public/share/get', {
      method: 'POST',
      body: JSON.stringify({
        share_id: shareId,
        path,
        token,
      }),
    });
  },
};

export const usageApi = {
  async get(): Promise<UsageResp> {
    return request<UsageResp>('/api/fs/usage', { method: 'GET' });
  },
};

export const trashApi = {
  async list(path = '/', page = 1, perPage = 100): Promise<TrashListResp> {
    return request<TrashListResp>(
      `/api/fs/trash?path=${encodeURIComponent(path)}&page=${page}&per_page=${perPage}`,
      { method: 'GET' }
    );
  },

  async restore(trashId: string | number, path = '/', targetPath?: string): Promise<void> {
    return request<void>('/api/fs/trash/restore', {
      method: 'POST',
      body: JSON.stringify({
        path,
        trash_id: trashId,
        ...(targetPath ? { target_path: targetPath } : {}),
      }),
    });
  },

  async delete(trashId: string | number, path = '/'): Promise<void> {
    return request<void>('/api/fs/trash/delete', {
      method: 'POST',
      body: JSON.stringify({
        path,
        trash_id: trashId,
        confirm: true,
      }),
    });
  },
};

export const versionApi = {
  async list(path: string, page = 1, perPage = 100): Promise<VersionsResp> {
    return request<VersionsResp>(
      `/api/fs/versions?path=${encodeURIComponent(path)}&page=${page}&per_page=${perPage}`,
      { method: 'GET' }
    );
  },

  async restore(path: string, versionId: string | number, revision: number): Promise<void> {
    return request<void>('/api/fs/versions/restore', {
      method: 'POST',
      body: JSON.stringify({
        path,
        version_id: versionId,
        revision,
      }),
    });
  },
};

export const textApi = {
  async get(path: string): Promise<TextGetResp> {
    return request<TextGetResp>(`/api/fs/text?path=${encodeURIComponent(path)}`, {
      method: 'GET',
    });
  },

  async put(path: string, content: string, revision: number): Promise<TextPutResp> {
    return request<TextPutResp>('/api/fs/text', {
      method: 'PUT',
      body: JSON.stringify({
        path,
        content,
        revision,
      }),
    });
  },
};

