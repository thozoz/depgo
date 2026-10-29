import { describe, expect, it, beforeEach } from 'vitest';
import { t, getLanguage, setLanguage } from '../utils/i18n';
import { describeError } from '../utils/errors';
import { ApiError } from '../api/client';

describe('i18n language infrastructure', () => {
  beforeEach(() => {
    (globalThis as any).document = {
      documentElement: { lang: 'tr' },
    };
    setLanguage('tr');
  });

  it('defaults to Turkish', () => {
    expect(getLanguage()).toBe('tr');
    expect(t('files.upload')).toBe('Dosya Yükle');
  });

  it('switches to English properly and updates document.documentElement.lang', () => {
    setLanguage('en');
    expect(getLanguage()).toBe('en');
    expect(t('files.upload')).toBe('Upload File');
    expect(document.documentElement.lang).toBe('en');
  });

  it('interpolates parameters', () => {
    setLanguage('tr');
    expect(t('files.selected_count', { count: 3 })).toBe('3 öğe seçildi');

    setLanguage('en');
    expect(t('files.selected_count', { count: 5 })).toBe('5 selected');
  });

  it('integrates with describeError in multiple languages', () => {
    setLanguage('tr');
    expect(describeError(new ApiError(200, 400, 'username or password is incorrect'), 'fallback')).toBe(
      'Kullanıcı adı veya parola hatalı.'
    );

    setLanguage('en');
    expect(describeError(new ApiError(200, 400, 'username or password is incorrect'), 'fallback')).toBe(
      'Username or password is incorrect.'
    );
  });
});
