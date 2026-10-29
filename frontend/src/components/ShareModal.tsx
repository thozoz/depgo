import React, { useState, useEffect } from 'react';
import { FileItem, ShareResp } from '../types';
import { shareApi } from '../api/client';
import { Modal } from './Modal';
import { ShareIcon, CopyIcon, CheckIcon } from './Icons';
import { copyText } from '../utils/clipboard';
import { describeShareError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

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
  const { t } = useTranslation();
  const [slug, setSlug] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [removePassword, setRemovePassword] = useState(false);
  const [expireHours, setExpireHours] = useState<number>(0);
  const [allowPreview, setAllowPreview] = useState(true);
  const [allowDownload, setAllowDownload] = useState(true);
  const [burnAfterRead, setBurnAfterRead] = useState(false);
  const [createdShare, setCreatedShare] = useState<ShareResp | null>(null);
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setError(null);
      setCopied(false);
      setLoading(false);
      setRemovePassword(false);
      if (existingShare) {
        setCreatedShare(existingShare);
        setSlug(existingShare.share_id);
        setName(existingShare.name);
        setPassword('');
        setAllowPreview(existingShare.allow_preview);
        setAllowDownload(existingShare.allow_download);
        setBurnAfterRead(existingShare.burn_after_read);
      } else if (item) {
        setCreatedShare(null);
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
        setBurnAfterRead(false);
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
          password: removePassword ? undefined : password || undefined,
          remove_password: removePassword || undefined,
          allow_preview: allowPreview,
          allow_download: allowDownload,
          burn_after_read: burnAfterRead,
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
          burn_after_read: burnAfterRead,
        });
      }

      setCreatedShare(result);
      if (onSuccess) onSuccess();
    } catch (err: unknown) {
      setError(describeShareError(err, 'Paylaşım işlemi başarısız'));
    } finally {
      setLoading(false);
    }
  };

  const handleCopyLink = async () => {
    if (!createdShare) return;
    const shareUrl = `${window.location.origin}/s/${createdShare.share_id}`;
    await copyText(shareUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const shareUrl = createdShare ? `${window.location.origin}/s/${createdShare.share_id}` : '';

  // Success view (when created or updated)
  if (createdShare && !existingShare) {
    return (
      <Modal
        isOpen={isOpen}
        onClose={onClose}
        title={t('share.ready_title')}
        icon={<CheckIcon className="w-5 h-5 text-accent" />}
      >
        <div className="space-y-4">
          <p className="text-sm text-charcoal-600">
            Aşağıdaki bağlantıyı kopyalayarak paylaşabilirsiniz.
          </p>

          <div className="flex items-center space-x-2">
            <input
              type="text"
              readOnly
              value={shareUrl}
              className="flex-1 px-3 py-2 bg-warm-50 border border-charcoal-200 rounded-lg text-xs font-mono text-charcoal-800 select-all"
            />
            <button
              type="button"
              onClick={handleCopyLink}
              className="flex items-center space-x-1.5 px-3 py-2 bg-accent hover:bg-accent-hover text-white text-xs font-semibold rounded-lg shadow-xs transition-colors flex-shrink-0"
            >
              {copied ? <CheckIcon className="w-4 h-4" /> : <CopyIcon className="w-4 h-4" />}
              <span>{copied ? t('common.copied') : t('common.copy')}</span>
            </button>
          </div>

          <div className="p-3 bg-warm-100 rounded-lg text-xs text-charcoal-600 space-y-1">
            <div className="flex justify-between">
              <span>Bağlantı:</span>
              <span className="font-mono text-charcoal-800 font-semibold">{createdShare.share_id}</span>
            </div>
            <div className="flex justify-between">
              <span>Güvenlik:</span>
              <span className="font-semibold text-charcoal-800">
                {createdShare.has_password ? 'Parolalı' : 'Açık Paylaşım'}
              </span>
            </div>
            {createdShare.burn_after_read && (
              <div className="flex justify-between text-yellow-700">
                <span>Mod:</span>
                <span className="font-semibold">Tek Okumalık</span>
              </div>
            )}
          </div>

          <div className="flex justify-end pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-semibold bg-accent hover:bg-accent-hover text-white rounded-lg shadow-xs transition-colors"
            >
              {t('common.close')}
            </button>
          </div>
        </div>
      </Modal>
    );
  }

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={existingShare ? 'Paylaşımı Düzenle' : 'Paylaşım Bağlantısı Oluştur'}
      icon={<ShareIcon className="w-5 h-5 text-accent" />}
      maxWidthClass="max-w-md"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="share-name-input" className="block text-xs font-semibold text-charcoal-700 mb-1">
            Paylaşım Başlığı
          </label>
          <input
            id="share-name-input"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Başlık girin"
            className="w-full px-3 py-2 border border-charcoal-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
            disabled={loading}
          />
        </div>

        <div>
          <label htmlFor="share-slug-input" className="block text-xs font-semibold text-charcoal-700 mb-1">
            {t('share.slug_label')}
          </label>
          <div className="flex items-center">
            <span className="px-2.5 py-2 bg-warm-100 border border-r-0 border-charcoal-300 rounded-l-lg text-xs font-mono text-charcoal-500 select-none">
              /s/
            </span>
            <input
              id="share-slug-input"
              type="text"
              value={slug}
              onChange={(e) => setSlug(e.target.value.toLowerCase())}
              placeholder="ornek-dosya"
              className="flex-1 px-3 py-2 border border-charcoal-300 rounded-r-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
              disabled={loading}
            />
          </div>
        </div>

        <div>
          <label htmlFor="share-password-input" className="block text-xs font-semibold text-charcoal-700 mb-1">
            {t('share.password_label')}
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
            className="w-full px-3 py-2 border border-charcoal-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent disabled:bg-charcoal-50 disabled:text-charcoal-400"
            disabled={loading || removePassword}
          />
          {existingShare?.has_password && (
            <label className="mt-2 flex items-center space-x-2 text-xs text-charcoal-700 cursor-pointer">
              <input
                type="checkbox"
                checked={removePassword}
                onChange={(e) => {
                  setRemovePassword(e.target.checked);
                  if (e.target.checked) setPassword('');
                }}
                disabled={loading}
                className="rounded border-charcoal-300 text-accent focus:ring-accent/30"
              />
              <span>Parolayı kaldır (bağlantıyı bilen herkes açabilir)</span>
            </label>
          )}
        </div>

        {!existingShare && (
          <div>
            <label htmlFor="share-expire-select" className="block text-xs font-semibold text-charcoal-700 mb-1">
              Son Kullanma Süresi
            </label>
            <select
              id="share-expire-select"
              value={expireHours}
              onChange={(e) => setExpireHours(Number(e.target.value))}
              className="w-full px-3 py-2 border border-charcoal-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent bg-white"
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

          <label className="flex items-start space-x-2 text-xs text-charcoal-700 cursor-pointer">
            <input
              type="checkbox"
              checked={burnAfterRead}
              onChange={(e) => setBurnAfterRead(e.target.checked)}
              disabled={loading}
              className="mt-0.5 rounded text-accent focus:ring-accent"
            />
            <span>
              {t('share.burn_after_read')}
              <span className="block text-charcoal-500 font-normal">
                {t('share.burn_description')}
              </span>
            </span>
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
            {t('common.close')}
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 text-sm font-semibold bg-accent hover:bg-accent-hover text-white rounded-lg shadow-xs transition-colors disabled:opacity-50"
          >
            {loading ? t('common.saving') : existingShare ? 'Güncelle' : 'Paylaşım Oluştur'}
          </button>
        </div>
      </form>
    </Modal>
  );
};
