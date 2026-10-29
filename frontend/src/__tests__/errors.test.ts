import { describe, expect, it } from 'vitest';
import { ApiError } from '../api/client';
import { describeError, describeShareError } from '../utils/errors';

describe('describeError', () => {
  it('translates known server messages and never shows the English text', () => {
    expect(describeError(new ApiError(200, 400, 'username or password is incorrect'), 'x')).toBe('Kullanıcı adı veya parola hatalı.');
    expect(describeError(new ApiError(200, 403, 'password is incorrect'), 'x')).toBe('Parola hatalı.');
    expect(describeError(new ApiError(200, 400, 'share link already exists'), 'x')).toContain('zaten kullanılıyor');
  });

  it('explains a rate limit by status code, whatever the server wrote', () => {
    expect(describeError(new ApiError(200, 429, 'Too many unsuccessful sign-in attempts'), 'x')).toContain('Çok fazla');
  });

  it('translates move conflicts (409 or file exists) into a localized conflict warning', () => {
    expect(describeError(new ApiError(200, 409, 'conflict'), 'x')).toContain('409');
    expect(describeError(new ApiError(200, 403, 'file [abc.txt] exists'), 'x')).toContain('409');
  });

  it('keeps an unknown message instead of hiding it, and uses the fallback for non-errors', () => {
    expect(describeError(new ApiError(200, 500, 'disk is full'), 'x')).toBe('disk is full');
    expect(describeError(new Error('Önizleme çok büyük'), 'x')).toBe('Önizleme çok büyük');
    expect(describeError('boom', 'Beklenmeyen hata')).toBe('Beklenmeyen hata');
  });

  it('does not call a missing file a missing share', () => {
    expect(describeError(new ApiError(200, 404, 'object not found'), 'x')).toBe('object not found');
  });
});

describe('describeShareError', () => {
  it('maps a gone link and a used-up link to plain Turkish without leaking why', () => {
    expect(describeShareError(new ApiError(200, 404, 'share not found'), 'x')).toContain('bulunamadı');
    expect(describeShareError(new ApiError(200, 404, 'record not found'), 'x')).toContain('bulunamadı');
    expect(describeShareError(new ApiError(200, 410, 'share has been consumed'), 'x')).toContain('süresi dolmuş');
  });

  it('still translates a known message that happens to arrive with a share status', () => {
    expect(describeShareError(new ApiError(200, 404, 'share link already exists'), 'x')).toContain('zaten kullanılıyor');
  });
});
