import React, { useState, useEffect } from 'react';
import { PublicShareInfoResp, PublicShareObjResp } from '../types';
import { publicShareApi } from '../api/client';
import { formatBytes, formatDate, isInlineRasterImage, isPdf, getBreadcrumbs, joinPath, getPreviewEligibility } from '../utils/format';
import {
  FolderIcon,
  FileIcon,
  ImageIcon,
  PdfIcon,
  DownloadIcon,
  EyeIcon,
  LockIcon,
  ChevronRightIcon,
  SpinnerIcon,
} from './Icons';
import { PreviewModal } from './PreviewModal';

interface GuestShareViewProps {
  shareId: string;
}

export const GuestShareView: React.FC<GuestShareViewProps> = ({ shareId }) => {
  const [info, setInfo] = useState<PublicShareInfoResp | null>(null);
  const [currentPath, setCurrentPath] = useState('/');
  const [items, setItems] = useState<PublicShareObjResp[]>([]);
  const [totalItems, setTotalItems] = useState(0);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const perPage = 100;

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Password state
  const [password, setPassword] = useState('');
  const [authRequired, setAuthRequired] = useState(false);
  const [authLoading, setAuthLoading] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);

  // Preview state
  const [previewItem, setPreviewItem] = useState<PublicShareObjResp | null>(null);

  const fetchShareInfo = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await publicShareApi.getInfo(shareId);
      setInfo(data);

      if (data.has_password && !data.authed) {
        setAuthRequired(true);
        setLoading(false);
        return;
      }

      setAuthRequired(false);
      if (data.is_dir) {
        await loadDirectory(currentPath, 1);
      } else {
        // Single file share
        const single = await publicShareApi.get(shareId, '');
        setItems([single.item]);
        setTotalItems(1);
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Paylaşım bilgisi alınamadı');
    } finally {
      setLoading(false);
    }
  };

  const loadDirectory = async (path: string, targetPage = 1) => {
    try {
      setLoading(true);
      const listResp = await publicShareApi.list(shareId, path, targetPage, perPage);
      setItems(listResp.content || []);
      setTotalItems(listResp.total || 0);
      setHasMore(!!listResp.has_more);
      setPage(targetPage);
      setCurrentPath(path);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Klasör içeriği yüklenemedi');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchShareInfo();
  }, [shareId]);

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!password) {
      setAuthError('Lütfen parolayı girin');
      return;
    }

    try {
      setAuthLoading(true);
      setAuthError(null);
      await publicShareApi.auth(shareId, password);
      setAuthRequired(false);
      if (info?.is_dir) {
        await loadDirectory(currentPath, 1);
      } else {
        const single = await publicShareApi.get(shareId, '');
        setItems([single.item]);
        setTotalItems(1);
      }
    } catch (err: unknown) {
      setAuthError(err instanceof Error ? err.message : 'Parola hatalı');
    } finally {
      setAuthLoading(false);
    }
  };

  const handleOpenFolder = (item: PublicShareObjResp) => {
    const nextPath = joinPath(currentPath, item.name);
    loadDirectory(nextPath, 1);
  };

  const renderItemIcon = (item: PublicShareObjResp) => {
    if (item.is_dir) {
      return <FolderIcon className="w-5 h-5 text-accent flex-shrink-0" />;
    }
    if (isInlineRasterImage(item.name)) {
      return <ImageIcon className="w-5 h-5 text-blue-500 flex-shrink-0" />;
    }
    if (isPdf(item.name)) {
      return <PdfIcon className="w-5 h-5 text-red-500 flex-shrink-0" />;
    }
    return <FileIcon className="w-5 h-5 text-charcoal-400 flex-shrink-0" />;
  };

  const handleDownload = (item: PublicShareObjResp) => {
    if (item.download_url) {
      window.open(item.download_url, '_blank');
    }
  };

  // Password requirement screen
  if (authRequired) {
    return (
      <div className="min-h-screen bg-warm-50 flex items-center justify-center p-4">
        <div className="bg-white border border-charcoal-200 rounded-2xl shadow-xl p-8 max-w-sm w-full text-center">
          <div className="w-12 h-12 rounded-2xl bg-yellow-50 text-yellow-600 flex items-center justify-center mx-auto mb-4 border border-yellow-200">
            <LockIcon className="w-6 h-6" />
          </div>
          <h2 className="text-lg font-semibold text-charcoal-900 mb-1">Korumalı Paylaşım</h2>
          <p className="text-xs text-charcoal-500 mb-6">
            Bu içeriğe erişmek için lütfen erişim parolasını girin.
          </p>

          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Parola"
              autoFocus
              className="w-full px-3.5 py-2.5 border border-charcoal-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
              disabled={authLoading}
            />

            {authError && (
              <div className="p-2.5 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700 text-left">
                {authError}
              </div>
            )}

            <button
              type="submit"
              disabled={authLoading || !password}
              className="w-full py-2.5 bg-accent hover:bg-accent-hover text-white text-sm font-medium rounded-xl shadow-sm transition-colors disabled:opacity-50"
            >
              {authLoading ? 'Doğrulanıyor...' : 'Erişimi Aç'}
            </button>
          </form>
        </div>
      </div>
    );
  }

  // Error screen
  if (error && !info) {
    return (
      <div className="min-h-screen bg-warm-50 flex items-center justify-center p-4">
        <div className="bg-white border border-charcoal-200 rounded-2xl shadow-md p-8 max-w-md w-full text-center">
          <div className="w-12 h-12 rounded-full bg-red-50 text-red-600 flex items-center justify-center mx-auto mb-4">
            <LockIcon className="w-6 h-6" />
          </div>
          <h2 className="text-lg font-semibold text-charcoal-900 mb-2">Paylaşıma Erişilemedi</h2>
          <p className="text-sm text-charcoal-600 mb-6">{error}</p>
          <a
            href="/"
            className="inline-block px-4 py-2 bg-charcoal-800 text-white rounded-lg text-sm font-medium hover:bg-charcoal-900"
          >
            Ana Sayfaya Dön
          </a>
        </div>
      </div>
    );
  }

  const breadcrumbs = getBreadcrumbs(currentPath);
  const totalPages = Math.ceil(totalItems / perPage) || 1;

  return (
    <div className="min-h-screen bg-warm-50 flex flex-col">
      {/* Header */}
      <header className="h-14 bg-white border-b border-charcoal-200 px-4 sm:px-6 flex items-center justify-between sticky top-0 z-10">
        <div className="flex items-center space-x-2.5">
          <div className="w-8 h-8 rounded-lg bg-accent/15 text-accent flex items-center justify-center font-bold text-base">
            D
          </div>
          <span className="font-semibold text-charcoal-900 text-base">Depgo Paylaşımı</span>
        </div>
        {info && (
          <div className="text-xs text-charcoal-500 truncate max-w-xs sm:max-w-md">
            Paylaşılan: <span className="font-medium text-charcoal-800">{info.name}</span>
          </div>
        )}
      </header>

      {/* Main Container */}
      <main className="flex-1 max-w-5xl w-full mx-auto p-4 sm:p-6 space-y-4">
        {/* Breadcrumbs (for folder share) */}
        {info?.is_dir && (
          <nav className="flex items-center space-x-1 text-sm bg-white border border-charcoal-200 rounded-xl px-4 py-2 overflow-x-auto">
            {breadcrumbs.map((crumb, idx) => {
              const isLast = idx === breadcrumbs.length - 1;
              return (
                <React.Fragment key={crumb.path}>
                  {idx > 0 && <ChevronRightIcon className="w-3.5 h-3.5 text-charcoal-400 flex-shrink-0" />}
                  <button
                    type="button"
                    onClick={() => loadDirectory(crumb.path, 1)}
                    disabled={isLast}
                    className={`flex items-center space-x-1.5 px-2 py-1 rounded text-sm transition-colors flex-shrink-0 ${
                      isLast
                        ? 'font-medium text-charcoal-900 cursor-default bg-charcoal-100/50'
                        : 'text-charcoal-600 hover:text-charcoal-900 hover:bg-charcoal-100'
                    }`}
                  >
                    {idx === 0 && <FolderIcon className="w-4 h-4 text-accent flex-shrink-0" />}
                    <span>{idx === 0 ? info.name : crumb.name}</span>
                  </button>
                </React.Fragment>
              );
            })}
          </nav>
        )}

        {/* Content table */}
        {loading && items.length === 0 ? (
          <div className="py-20 text-center text-charcoal-500">
            <SpinnerIcon className="w-8 h-8 text-accent animate-spin mx-auto mb-2" />
            <span className="text-sm">İçerik yükleniyor...</span>
          </div>
        ) : items.length === 0 ? (
          <div className="py-16 text-center text-charcoal-400 bg-white border border-charcoal-200 rounded-xl">
            <FolderIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
            <p className="text-sm">Bu klasörde görüntülenecek öğe bulunamadı.</p>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="overflow-x-auto bg-white border border-charcoal-200 rounded-xl shadow-sm">
              <table className="w-full text-left text-sm">
                <thead className="bg-warm-100 border-b border-charcoal-200 text-xs font-medium text-charcoal-600 uppercase tracking-wider select-none">
                  <tr>
                    <th className="py-3 px-4">Ad</th>
                    <th className="py-3 px-4 w-28 hidden sm:table-cell">Boyut</th>
                    <th className="py-3 px-4 w-44 hidden md:table-cell">Tarih</th>
                    <th className="py-3 px-4 w-32 text-right">İşlem</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-charcoal-100">
                  {items.map((item) => {
                    const previewEligible = !item.is_dir && getPreviewEligibility(item.name, item.size).canPreview;

                    return (
                      <tr key={item.name} className="hover:bg-warm-50/70 transition-colors">
                        <td className="py-2.5 px-4">
                          <button
                            type="button"
                            onClick={() => {
                              if (item.is_dir) {
                                handleOpenFolder(item);
                              } else if (previewEligible && item.preview_url) {
                                setPreviewItem(item);
                              } else {
                                handleDownload(item);
                              }
                            }}
                            className="flex items-center space-x-2.5 text-left text-charcoal-800 hover:text-accent font-medium truncate max-w-xs sm:max-w-md"
                            title={item.name}
                          >
                            {renderItemIcon(item)}
                            <span className="truncate">{item.name}</span>
                          </button>
                        </td>
                        <td className="py-2.5 px-4 text-xs text-charcoal-500 hidden sm:table-cell">
                          {item.is_dir ? '-' : formatBytes(item.size)}
                        </td>
                        <td className="py-2.5 px-4 text-xs text-charcoal-500 hidden md:table-cell">
                          {formatDate(item.modified)}
                        </td>
                        <td className="py-2.5 px-4 text-right">
                          <div className="flex items-center justify-end space-x-1.5">
                            {previewEligible && item.preview_url && (
                              <button
                                type="button"
                                onClick={() => setPreviewItem(item)}
                                className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                                title="Önizle"
                              >
                                <EyeIcon className="w-4 h-4" />
                              </button>
                            )}
                            {!item.is_dir && item.download_url && (
                              <button
                                type="button"
                                onClick={() => handleDownload(item)}
                                className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                                title="İndir"
                              >
                                <DownloadIcon className="w-4 h-4" />
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {/* Pagination for public share */}
            {(totalItems > perPage || hasMore) && (
              <div className="flex items-center justify-between px-3 py-2 bg-white border border-charcoal-200 rounded-xl text-xs text-charcoal-600">
                <span>
                  Toplam {totalItems} öğeden {(page - 1) * perPage + 1} -{' '}
                  {Math.min(page * perPage, totalItems)} arası gösteriliyor
                </span>
                <div className="flex items-center space-x-2">
                  <button
                    type="button"
                    onClick={() => loadDirectory(currentPath, page - 1)}
                    disabled={page <= 1 || loading}
                    className="px-2.5 py-1 bg-white border border-charcoal-200 rounded hover:bg-warm-100 disabled:opacity-40"
                  >
                    Önceki
                  </button>
                  <span className="font-medium text-charcoal-800">
                    {page} / {totalPages}
                  </span>
                  <button
                    type="button"
                    onClick={() => loadDirectory(currentPath, page + 1)}
                    disabled={(!hasMore && page >= totalPages) || loading}
                    className="px-2.5 py-1 bg-white border border-charcoal-200 rounded hover:bg-warm-100 disabled:opacity-40"
                  >
                    Sonraki
                  </button>
                </div>
              </div>
            )}
          </div>
        )}
      </main>

      {/* Preview modal for guest (safely validated) */}
      {previewItem && (
        <PreviewModal
          isOpen={true}
          name={previewItem.name}
          size={previewItem.size}
          publicPreviewUrl={previewItem.preview_url}
          onClose={() => setPreviewItem(null)}
          onDownload={() => handleDownload(previewItem)}
        />
      )}
    </div>
  );
};
