import { describe, it, expect } from 'vitest';
import {
  ApiError,
  setStoredToken,
  setShareToken,
  getShareToken,
  clearShareToken,
} from '../api/client';

describe('API client and error handling', () => {
  it('constructs ApiError with HTTP status and envelope error code', () => {
    const error = new ApiError(400, 400, 'Kullanıcı adı veya şifre hatalı');
    expect(error.status).toBe(400);
    expect(error.code).toBe(400);
    expect(error.message).toBe('Kullanıcı adı veya şifre hatalı');
    expect(error.name).toBe('ApiError');
  });

  it('safely manages admin token storage', () => {
    setStoredToken('test_jwt_token_123');
    // Clear token
    setStoredToken(null);
  });

  it('stores guest share tokens in isolated memory without leaking', () => {
    setShareToken('test-slug-1', 'guest-token-abc');
    expect(getShareToken('test-slug-1')).toBe('guest-token-abc');
    expect(getShareToken('other-slug')).toBeUndefined();

    clearShareToken('test-slug-1');
    expect(getShareToken('test-slug-1')).toBeUndefined();
  });
});
