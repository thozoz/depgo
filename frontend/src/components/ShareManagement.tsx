import React, { useState, useEffect } from 'react';
import { ShareResp } from '../types';
import { shareApi } from '../api/client';
import { formatDate } from '../utils/format';
import { copyText } from '../utils/clipboard';
import {
  ShareIcon,
  CopyIcon,
  CheckIcon,
  EditIcon,
  TrashIcon,
  RefreshIcon,
  LockIcon,
  FolderIcon,
  FileIcon,
  SpinnerIcon,
} from './Icons';
import { ShareModal } from './ShareModal';

export const ShareManagement: React.FC = () => {
  const [shares, setShares] = useState<ShareResp[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const perPage = 20;

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [editingShare, setEditingShare] = useState<ShareResp | null>(null);

  const fetchShares = async (targetPage = page) => {
    try {
      setLoading(true);
      setError(null);
      const resp = await shareApi.list(targetPage, perPage);
      setShares(resp.content || []);
      setTotal(resp.total || 0);
      setPage(targetPage);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Paylaşımlar yüklenemedi');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchShares(1);
  }, []);

  const handleCopy = async (shareId: string) => {
    const url = `${window.location.origin}/s/${shareId}`;
    await copyText(url);
    setCopiedId(shareId);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const handleDisable = async (share: ShareResp) => {
    if (!share.enabled) {
      alert('Mevcut AList backend API sözleşmesinde kapatılmış bir paylaşımı yeniden açma (re-enable) endpoint\'i bulunmamaktadır. Yeniden paylaşmak için yeni bir bağlantı oluşturunuz.');
      return;
    }
    if (!window.confirm(`"${share.name}" paylaşımını devre dışı bırakmak istediğinize emin misiniz? (AList backend'de kapatılan paylaşım yeniden açılamaz)`)) {
      return;
    }
    try {
      await shareApi.disable(share.share_id);
      fetchShares(page);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Paylaşım kapatılamadı');
    }
  };

  const handleDelete = async (shareId: string) => {
    if (!window.confirm('Bu paylaşım bağlantısını tamamen silmek istediğinize emin misiniz?')) {
      return;
    }
    try {
      await shareApi.delete(shareId);
      fetchShares(page);
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : 'Paylaşım silinemedi');
    }
  };

  const totalPages = Math.ceil(total / perPage) || 1;

  return (
    <div className="space-y-4">
      {/* Header bar */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-charcoal-900">Paylaşım Bağlantıları</h2>
          <p className="text-xs text-charcoal-500">
            Dış kullanıcılar için oluşturulan dosya ve klasör bağlantılarını yönetin.
          </p>
        </div>
        <button
          type="button"
          onClick={() => fetchShares(page)}
          disabled={loading}
          className="flex items-center space-x-1 px-3 py-1.5 bg-white border border-charcoal-200 hover:bg-warm-100 rounded-lg text-xs font-medium text-charcoal-700 transition-colors"
          title="Listeyi Yenile"
        >
          <RefreshIcon className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          <span>Yenile</span>
        </button>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-xs rounded-xl">
          {error}
        </div>
      )}

      {loading && shares.length === 0 ? (
        <div className="py-16 text-center text-charcoal-500">
          <SpinnerIcon className="w-8 h-8 text-accent animate-spin mx-auto mb-2" />
          <span className="text-sm">Paylaşımlar yükleniyor...</span>
        </div>
      ) : shares.length === 0 ? (
        <div className="py-16 text-center text-charcoal-400 bg-white border border-charcoal-200 rounded-xl">
          <ShareIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
          <p className="text-sm font-medium text-charcoal-600">Henüz aktif bir paylaşım bağlantısı yok</p>
          <p className="text-xs text-charcoal-400 mt-1">
            Dosyalar sekmesinden bir dosya veya klasör seçip "Paylaş" butonuna tıklayarak bağlantı oluşturabilirsiniz.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="overflow-x-auto bg-white border border-charcoal-200 rounded-xl">
            <table className="w-full text-left text-sm">
              <thead className="bg-warm-100 border-b border-charcoal-200 text-xs font-medium text-charcoal-600 uppercase tracking-wider select-none">
                <tr>
                  <th className="py-3 px-4">Bağlantı & Ad</th>
                  <th className="py-3 px-4 hidden md:table-cell">Kök Yol</th>
                  <th className="py-3 px-4 hidden sm:table-cell">Güvenlik</th>
                  <th className="py-3 px-4 hidden lg:table-cell">Son Kullanma</th>
                  <th className="py-3 px-4 hidden sm:table-cell">Görüntüleme</th>
                  <th className="py-3 px-4">Durum</th>
                  <th className="py-3 px-4 text-right">İşlemler</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-charcoal-100">
                {shares.map((share) => (
                  <tr key={share.id} className="hover:bg-warm-50/70 transition-colors">
                    <td className="py-3 px-4">
                      <div className="flex flex-col">
                        <div className="flex items-center space-x-1.5 font-medium text-charcoal-900">
                          {share.is_dir ? (
                            <FolderIcon className="w-4 h-4 text-accent flex-shrink-0" />
                          ) : (
                            <FileIcon className="w-4 h-4 text-charcoal-400 flex-shrink-0" />
                          )}
                          <span className="truncate max-w-xs">{share.name}</span>
                        </div>
                        <div className="flex items-center space-x-1.5 mt-0.5">
                          <span className="font-mono text-xs text-accent">/s/{share.share_id}</span>
                          <button
                            type="button"
                            onClick={() => handleCopy(share.share_id)}
                            className="p-1 text-charcoal-400 hover:text-charcoal-700 rounded"
                            title="Bağlantıyı Kopyala"
                          >
                            {copiedId === share.share_id ? (
                              <CheckIcon className="w-3.5 h-3.5 text-accent" />
                            ) : (
                              <CopyIcon className="w-3.5 h-3.5" />
                            )}
                          </button>
                        </div>
                      </div>
                    </td>
                    <td className="py-3 px-4 font-mono text-xs text-charcoal-500 hidden md:table-cell truncate max-w-xs">
                      {share.root_path}
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-600 hidden sm:table-cell">
                      {share.has_password ? (
                        <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded bg-yellow-50 text-yellow-700 border border-yellow-200">
                          <LockIcon className="w-3 h-3" />
                          <span>Parolalı</span>
                        </span>
                      ) : (
                        <span className="text-charcoal-400">Genel</span>
                      )}
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-500 hidden lg:table-cell">
                      {share.expires_at ? formatDate(share.expires_at) : 'Süresiz'}
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-600 hidden sm:table-cell">
                      {share.view_count} kez
                    </td>
                    <td className="py-3 px-4">
                      <span
                        className={`inline-block px-2 py-0.5 rounded-full text-xs font-medium ${
                          share.enabled
                            ? 'bg-accent/15 text-accent-dark'
                            : 'bg-charcoal-100 text-charcoal-500'
                        }`}
                      >
                        {share.enabled ? 'Aktif' : 'Devre Dışı'}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-right">
                      <div className="flex items-center justify-end space-x-1">
                        <button
                          type="button"
                          onClick={() => setEditingShare(share)}
                          className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                          title="Düzenle"
                        >
                          <EditIcon className="w-4 h-4" />
                        </button>
                        {share.enabled && (
                          <button
                            type="button"
                            onClick={() => handleDisable(share)}
                            className="text-xs px-2 py-1 rounded border border-charcoal-200 text-charcoal-600 hover:bg-charcoal-100 transition-colors"
                            title="Bağlantıyı Devre Dışı Bırak"
                          >
                            Kapat
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => handleDelete(share.share_id)}
                          className="p-1.5 text-charcoal-500 hover:text-red-600 hover:bg-red-50 rounded"
                          title="Sil"
                        >
                          <TrashIcon className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination controls */}
          {total > perPage && (
            <div className="flex items-center justify-between px-2 py-1 text-xs text-charcoal-600">
              <span>
                Toplam {total} paylaşımdan {(page - 1) * perPage + 1} - {Math.min(page * perPage, total)} arası gösteriliyor
              </span>
              <div className="flex items-center space-x-2">
                <button
                  type="button"
                  onClick={() => fetchShares(page - 1)}
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
                  onClick={() => fetchShares(page + 1)}
                  disabled={page >= totalPages || loading}
                  className="px-2.5 py-1 bg-white border border-charcoal-200 rounded hover:bg-warm-100 disabled:opacity-40"
                >
                  Sonraki
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Edit modal */}
      {editingShare && (
        <ShareModal
          isOpen={true}
          item={null}
          existingShare={editingShare}
          onClose={() => setEditingShare(null)}
          onSuccess={() => {
            setEditingShare(null);
            fetchShares(page);
          }}
        />
      )}
    </div>
  );
};
