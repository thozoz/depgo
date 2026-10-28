import React, { useState, useEffect } from 'react';
import { fsApi, ApiError } from '../api/client';
import { formatBytes, isInlineRasterImage, isPdf, isSvgOrHtml, MAX_PREVIEW_SIZE_BYTES } from '../utils/format';
import { CloseIcon, DownloadIcon, SpinnerIcon, EyeIcon } from './Icons';

interface PreviewModalProps {
  isOpen: boolean;
  name: string;
  size?: number;
  rawUrl?: string;
  publicPreviewUrl?: string;
  onClose: () => void;
  onDownload: () => void;
}

export const PreviewModal: React.FC<PreviewModalProps> = ({
  isOpen,
  name,
  size,
  rawUrl,
  publicPreviewUrl,
  onClose,
  onDownload,
}) => {
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const titleId = React.useId();

  useEffect(() => {
    if (!isOpen) {
      if (blobUrl && blobUrl.startsWith('blob:')) {
        URL.revokeObjectURL(blobUrl);
      }
      setBlobUrl(null);
      setError(null);
      setLoading(false);
      return;
    }

    // SVG / HTML active content is strictly download-only
    if (isSvgOrHtml(name)) {
      setError('Güvenlik nedeniyle SVG ve HTML dosyaları tarayıcıda önizlenmez. Lütfen dosyayı indirin.');
      setLoading(false);
      return;
    }

    const isRaster = isInlineRasterImage(name);
    const isPdfDoc = isPdf(name);

    if (!isRaster && !isPdfDoc) {
      setError('Bu dosya türü için önizleme desteklenmiyor.');
      setLoading(false);
      return;
    }

    if (size !== undefined && size > MAX_PREVIEW_SIZE_BYTES) {
      setError(`Dosya boyutu 20 MB sınırını aşıyor (${formatBytes(size)}). Lütfen dosyayı indirin.`);
      setLoading(false);
      return;
    }

    const targetUrl = rawUrl || publicPreviewUrl;
    if (!targetUrl) {
      setError('Önizleme adresi bulunamadı.');
      return;
    }

    const isPublic = !!publicPreviewUrl && !rawUrl;
    const abortController = new AbortController();

    setLoading(true);
    setError(null);

    // Revoke previous blob if any
    if (blobUrl && blobUrl.startsWith('blob:')) {
      URL.revokeObjectURL(blobUrl);
      setBlobUrl(null);
    }

    fsApi
      .fetchAuthorizedBlob(targetUrl, name, MAX_PREVIEW_SIZE_BYTES, abortController.signal, isPublic)
      .then((blob) => {
        if (abortController.signal.aborted) return;
        const objectUrl = URL.createObjectURL(blob);
        setBlobUrl(objectUrl);
        setLoading(false);
      })
      .catch((err: unknown) => {
        if (abortController.signal.aborted) return;
        if (err instanceof ApiError) {
          setError(err.message);
        } else {
          setError(err instanceof Error ? err.message : 'Önizleme yüklenemedi');
        }
        setLoading(false);
      });

    return () => {
      abortController.abort();
    };
  }, [isOpen, name, rawUrl, publicPreviewUrl, size]);

  // Clean up blob URL on unmount
  useEffect(() => {
    return () => {
      if (blobUrl && blobUrl.startsWith('blob:')) {
        URL.revokeObjectURL(blobUrl);
      }
    };
  }, [blobUrl]);

  // Handle Escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        e.preventDefault();
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const isRaster = isInlineRasterImage(name);
  const isPdfDoc = isPdf(name);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4"
    >
      <div
        className="fixed inset-0 bg-charcoal-900/60 backdrop-blur-sm transition-opacity"
        onClick={onClose}
        aria-hidden="true"
      />
      <div className="relative bg-white rounded-2xl shadow-2xl border border-charcoal-200 max-w-4xl w-full h-[90vh] z-10 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-charcoal-200 bg-warm-100 flex-shrink-0">
          <div className="flex items-center space-x-2 truncate">
            <EyeIcon className="w-5 h-5 text-accent flex-shrink-0" />
            <span id={titleId} className="font-semibold text-charcoal-900 text-sm truncate" title={name}>
              {name}
            </span>
            {size !== undefined && (
              <span className="text-xs text-charcoal-500 flex-shrink-0">({formatBytes(size)})</span>
            )}
          </div>
          <div className="flex items-center space-x-2">
            <button
              type="button"
              onClick={onDownload}
              className="flex items-center space-x-1.5 px-3 py-1.5 bg-accent hover:bg-accent-hover text-white text-xs font-medium rounded-lg shadow-sm transition-colors"
              title="Dosyayı İndir"
            >
              <DownloadIcon className="w-3.5 h-3.5" />
              <span>İndir</span>
            </button>
            <button
              type="button"
              onClick={onClose}
              className="p-1 text-charcoal-400 hover:text-charcoal-700 rounded-lg transition-colors"
              aria-label="Kapat"
            >
              <CloseIcon className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Content viewer */}
        <div className="flex-1 bg-charcoal-50 p-4 flex items-center justify-center overflow-auto">
          {loading && (
            <div className="flex flex-col items-center space-y-2 text-charcoal-500">
              <SpinnerIcon className="w-8 h-8 text-accent animate-spin" />
              <span className="text-sm">Önizleme güvenli şekilde hazırlanıyor...</span>
            </div>
          )}

          {error && (
            <div className="max-w-md p-6 bg-white border border-charcoal-200 rounded-xl text-center space-y-3 shadow-sm">
              <p className="text-sm text-charcoal-700">{error}</p>
              <button
                type="button"
                onClick={onDownload}
                className="inline-flex items-center space-x-2 px-4 py-2 bg-accent hover:bg-accent-hover text-white text-sm font-medium rounded-lg shadow-sm transition-colors"
              >
                <DownloadIcon className="w-4 h-4" />
                <span>Dosyayı İndir</span>
              </button>
            </div>
          )}

          {!loading && !error && blobUrl && isRaster && (
            <div className="flex items-center justify-center w-full h-full">
              <img
                src={blobUrl}
                alt={name}
                className="max-w-full max-h-full object-contain rounded-lg shadow-sm"
              />
            </div>
          )}

          {!loading && !error && blobUrl && isPdfDoc && (
            <div className="w-full h-full flex flex-col">
              <object
                data={blobUrl}
                type="application/pdf"
                className="w-full h-full rounded-lg border border-charcoal-200"
              >
                <div className="p-6 text-center bg-white rounded-lg border border-charcoal-200 my-auto">
                  <p className="text-sm text-charcoal-600 mb-3">
                    Tarayıcınız yerleşik PDF görüntüleyiciyi desteklemiyor.
                  </p>
                  <button
                    type="button"
                    onClick={onDownload}
                    className="inline-flex items-center space-x-1.5 px-4 py-2 bg-accent hover:bg-accent-hover text-white text-xs font-medium rounded-lg"
                  >
                    <DownloadIcon className="w-3.5 h-3.5" />
                    <span>PDF'i İndir</span>
                  </button>
                </div>
              </object>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
