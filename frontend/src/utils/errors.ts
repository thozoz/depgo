import { ApiError } from '../api/client';

// Every user-facing text for a failed request comes from here. The backend never
// localises: it answers with a status code and a stable English message, and this
// module turns them into what the visitor reads. When the site gets its Turkish
// and English switch, this is the one table that needs translating.
const BY_MESSAGE: Record<string, string> = {
  'username or password is incorrect': 'Kullanıcı adı veya parola hatalı.',
  'password is incorrect': 'Parola hatalı.',
  'share link already exists': 'Bu bağlantı adı zaten kullanılıyor. Başka bir ad seçin.',
};

const TOO_MANY_ATTEMPTS = 'Çok fazla hatalı deneme yapıldı. Lütfen biraz sonra tekrar deneyin.';
const SHARE_NOT_FOUND =
  'Bu paylaşım bulunamadı. Silinmiş ya da kapatılmış olabilir; tek okumalık bir bağlantı ise zaten kullanılmış olabilir.';
const SHARE_EXPIRED = 'Bu paylaşımın süresi dolmuş veya kullanım hakkı bitmiş.';

export function describeError(err: unknown, fallback: string): string {
  if (err instanceof ApiError) {
    const known = BY_MESSAGE[err.message];
    if (known) return known;
    if (err.code === 429) return TOO_MANY_ATTEMPTS;
    return err.message;
  }
  return err instanceof Error ? err.message : fallback;
}

// describeShareError adds the answers that only mean something for a share link:
// a link that is gone, closed, expired or already used looks the same from
// outside, and none of them should say which it was.
export function describeShareError(err: unknown, fallback: string): string {
  if (err instanceof ApiError && !BY_MESSAGE[err.message]) {
    if (err.code === 404) return SHARE_NOT_FOUND;
    if (err.code === 410) return SHARE_EXPIRED;
  }
  return describeError(err, fallback);
}
