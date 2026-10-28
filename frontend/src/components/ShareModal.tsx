import React, { useState, useEffect } from 'react';
import { FileItem, ShareResp } from '../types';
import { shareApi } from '../api/client';
import { Modal } from './Modal';
import { ShareIcon, CopyIcon, CheckIcon } from './Icons';

interface ShareModalProps {
  isOpen: boolean;
  item: FileItem | null;
  existingShare?: ShareResp | null;
  onClose: () => void;
  onSuccess?: () => void;
}

export const ShareModal: React.FC<ShareModalProps> = ({
  isOpen,
  item,
  existingShare,
  onClose,
  onSuccess,
}) => {
  const [slug, setSlug] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [expireHours, setExpireHours] = useState<number>(0);
  const [allowPreview, setAllowPreview] = useState(true);
  const [allowDownload, setAllowDownload] = useState(true);
  const [createdShare, setCreatedShare] = useState<ShareResp | null>(null);
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setError(null);
      setCopied(false);
      setLoading(false);
      if (existingShare) {
        setCreatedShare(existingShare);
        setSlug(existingShare.share_id);
        setName(existingShare.name);
        setPassword('');
        setAllowPreview(existingShare.allow_preview);
        setAllowDownload(existingShare.allow_download);
      } else if (item) {
        setCreatedShare(null);
        // Default slug from name
        const autoSlug = item.name
          .toLowerCase()
          .replace(/[^a-z0-9_-]/g, '-')
          .replace(/-+/g, '-')
          .slice(0, 20);
        setSlug(autoSlug);
        setName(item.name);
        setPassword('');
        setExpireHours(0);
        setAllowPreview(true);
        setAllowDownload(true);
      }
    }
  }, [isOpen, item, existingShare]);

  if (!isOpen || (!item && !existingShare)) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (slug && !/^[A-Za-z0-9_-]{1,32}$/.test(slug)) {
      setError('Özel bağlantı adı (slug) 1-32 karakter uzunluğunda olmalı ve yalnızca harf, rakam, alt çizgi veya tire içermelidir.');
      return;
    }

    try {
      setLoading(true);
      setError(null);

      let result: ShareResp;
      if (existingShare) {
        result = await shareApi.update({
          share_id: existingShare.share_id,
          new_share_id: slug || undefined,
          name: name.trim() || undefined,
          password: password || undefined,
          allow_preview: allowPreview,
          allow_download: allowDownload,
        });
      } else {
        const fullItemPath = item!.virtual_path || item!.path || item!.name;
        result = await shareApi.create({
          path: fullItemPath,
          share_id: slug.trim() || undefined,
          name: name.trim() || item!.name,
          password: password || undefined,
          expire_hours: expireHours > 0 ? expireHours : undefined,
          allow_preview: allowPreview,
          allow_download: allowDownload,
        });
      }

      setCreatedShare(result);
      if (onSuccess) onSuccess();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Paylaşım işlemi başarısız');
    } finally {
      setLoading(false);
    }
  };

  const getShareUrl = (s: ShareResp) => {
    return `${window.location.origin}/s/${s.share_id}`;
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={existingShare ? 'Paylaşım Bağlantısını Düzenle' : 'Paylaşım Bağlantısı Oluştur'}
      icon={<ShareIcon className="w-5 h-5 text-accent" />}
      maxWidthClass="max-w-lg"
    >
      {createdShare && (
        <div className="p-4 bg-accent/10 border border-accent/20 rounded-xl mb-5 space-y-2">
          <span className="text-xs font-semibold text-accent-dark uppercase tracking-wide">
            Paylaşım Bağlantınız Hazır
          </span>
          <div className="flex items-center space-x-2">
            <input
              type="text"
              readOnly
              value={getShareUrl(createdShare)}
              className="w-full text-xs font-mono bg-white border border-charcoal-200 rounded-lg px-2.5 py-1.5 select-all"
            />
            <button
              type="button"
              onClick={() => copyToClipboard(getShareUrl(createdShare))}
              className="flex items-center space-x-1 px-3 py-1.5 bg-accent hover:bg-accent-hover text-white text-xs font-medium rounded-lg shadow-sm transition-colors flex-shrink-0"
            >
              {copied ? <CheckIcon className="w-3.5 h-3.5" /> : <CopyIcon className="w-3.5 h-3.5" />}
              <span>{copied ? 'Kopyalandı' : 'Kopyala'}</span>
            </button>
          </div>
          <p className="text-[11px] text-charcoal-500">
            Bu bağlantıyı alan kullanıcılar hesaba giriş yapmadan dosyayı görüntüleyebilir veya indirebilir.
          </p>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="share-name-input" className="block text-xs font-medium text-charcoal-700 mb-1">
            Paylaşım Başlığı
          </label>
          <input
            id="share-name-input"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Örn. Paylaşılan Dosya"
            className="w-full px-3 py-2 border border-charcoal-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
            disabled={loading}
          />
        </div>

        <div>
          <label htmlFor="share-slug-input" className="block text-xs font-medium text-charcoal-700 mb-1">
            Özel Bağlantı Adı (Slug)
          </label>
          <div className="flex items-center">
            <span className="text-xs text-charcoal-400 bg-charcoal-100 border border-r-0 border-charcoal-200 px-2.5 py-2 rounded-l-lg select-none">
              /s/
            </span>
            <input
              id="share-slug-input"
              type="text"
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              placeholder="ozel-ad (isteğe bağlı)"
              className="w-full px-3 py-2 border border-charcoal-200 rounded-r-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent font-mono text-xs"
              disabled={loading}
            />
          </div>
          <span className="text-[11px] text-charcoal-400 mt-1 block">
            Harf, rakam, tire veya alt çizgi (1-32 karakter). Boş bırakılırsa otomatik üretilir.
          </span>
        </div>

        <div>
          <label htmlFor="share-password-input" className="block text-xs font-medium text-charcoal-700 mb-1">
            Erişim Parolası (İsteğe bağlı)
          </label>
          <input
            id="share-password-input"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder={
              existingShare?.has_password
                ? 'Yeni parola girin (mevcut parolayı korumak için boş bırakın)'
                : 'Parola belirleyin'
            }
            className="w-full px-3 py-2 border border-charcoal-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
            disabled={loading}
          />
        </div>

        {!existingShare && (
          <div>
            <label htmlFor="share-expire-select" className="block text-xs font-medium text-charcoal-700 mb-1">
              Son Kullanma Süresi
            </label>
            <select
              id="share-expire-select"
              value={expireHours}
              onChange={(e) => setExpireHours(Number(e.target.value))}
              className="w-full px-3 py-2 border border-charcoal-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent bg-white"
              disabled={loading}
            >
              <option value={0}>Süresiz</option>
              <option value={1}>1 Saat</option>
              <option value={24}>24 Saat (1 Gün)</option>
              <option value={72}>3 Gün</option>
              <option value={168}>7 Gün (1 Hafta)</option>
              <option value={720}>30 Gün (1 Ay)</option>
            </select>
          </div>
        )}

        <div className="pt-2 border-t border-charcoal-100 space-y-2">
          <label className="flex items-center space-x-2 text-xs text-charcoal-700 cursor-pointer">
            <input
              type="checkbox"
              checked={allowPreview}
              onChange={(e) => setAllowPreview(e.target.checked)}
              className="rounded text-accent focus:ring-accent"
            />
            <span>Tarayıcı içi önizlemeye izin ver</span>
          </label>

          <label className="flex items-center space-x-2 text-xs text-charcoal-700 cursor-pointer">
            <input
              type="checkbox"
              checked={allowDownload}
              onChange={(e) => setAllowDownload(e.target.checked)}
              className="rounded text-accent focus:ring-accent"
            />
            <span>İndirmeye izin ver</span>
          </label>
        </div>

        {error && (
          <div className="p-2.5 rounded-lg bg-red-50 text-red-700 text-xs border border-red-200">
            {error}
          </div>
        )}

        <div className="flex justify-end space-x-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            disabled={loading}
            className="px-3.5 py-2 text-sm font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg transition-colors"
          >
            Kapat
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg shadow-sm transition-colors disabled:opacity-50"
          >
            {loading ? 'Kaydediliyor...' : existingShare ? 'Güncelle' : 'Paylaşım Oluştur'}
          </button>
        </div>
      </form>
    </Modal>
  );
};
