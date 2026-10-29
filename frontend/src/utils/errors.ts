import { ApiError } from '../api/client';
import { t } from './i18n';

// Every user-facing text for a failed request comes from here. The backend never
// localises: it answers with a status code and a stable English message, and this
// module turns them into what the visitor reads using the i18n translation system.
const BY_MESSAGE: Record<string, string> = {
  'username or password is incorrect': 'error.incorrect_credentials',
  'password is incorrect': 'error.incorrect_password',
  'share link already exists': 'error.share_already_exists',
};

export function describeError(err: unknown, fallback: string): string {
  if (err instanceof ApiError) {
    const key = BY_MESSAGE[err.message];
    if (key) return t(key);
    if (err.code === 429) return t('error.too_many_attempts');
    return err.message;
  }
  return err instanceof Error ? err.message : fallback;
}

// describeShareError adds the answers that only mean something for a share link:
// a link that is gone, closed, expired or already used looks the same from
// outside, and none of them should say which it was.
export function describeShareError(err: unknown, fallback: string): string {
  if (err instanceof ApiError && !BY_MESSAGE[err.message]) {
    if (err.code === 404) return t('error.share_not_found');
    if (err.code === 410) return t('error.share_expired');
  }
  return describeError(err, fallback);
}
