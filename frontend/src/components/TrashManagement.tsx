import React, { useState, useEffect, useCallback } from 'react';
import { TrashItem } from '../types';
import { trashApi, ApiError } from '../api/client';
import { formatBytes, formatDate } from '../utils/format';
import { Modal } from './Modal';
import {
  TrashIcon,
  RestoreIcon,
  FolderIcon,
  FileIcon,
  RefreshIcon,
  SpinnerIcon,
  AlertTriangleIcon,
} from './Icons';
import { describeError } from '../utils/errors';

export const TrashManagement: React.FC = () => {
  const [items, setItems] = useState<TrashItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const perPage = 50;

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Actions state
  const [actionLoadingId, setActionLoadingId] = useState<string | number | null>(null);
  const [deleteConfirmItem, setDeleteConfirmItem] = useState<TrashItem | null>(null);
  const [collisionItem, setCollisionItem] = useState<TrashItem | null>(null);
  const [customTargetPath, setCustomTargetPath] = useState('');
  const [collisionError, setCollisionError] = useState<string | null>(null);

  const fetchTrash = useCallback(
    async (targetPage = 1) => {
      try {
        setLoading(true);
        setError(null);
        const data = await trashApi.list('/', targetPage, perPage);
        setItems(data.content || []);
        setTotal(data.total || 0);
        setPage(targetPage);
      } catch (err: unknown) {
        setError(describeError(err, 'Çöp kutusu listesi alınamadı'));
      } finally {
        setLoading(false);
      }
    },
    []
  );

  useEffect(() => {
    fetchTrash(1);
  }, [fetchTrash]);

  const handleRestore = async (item: TrashItem, customPath?: string) => {
    try {
      setActionLoadingId(item.trash_id);
      setError(null);
      await trashApi.restore(item.trash_id, '/', customPath);
      setCollisionItem(null);
      setCustomTargetPath('');
      setCollisionError(null);
      await fetchTrash(page);
    } catch (err: unknown) {
      if (err instanceof ApiError && (err.status === 409 || err.code === 409 || err.message.toLowerCase().includes('exist'))) {
        // 409 collision: original path occupied! Open target_path prompt
        setCollisionItem(item);
        setCustomTargetPath(item.original_path + '.restored');
        setCollisionError('Orijinal konumda aynı isimde bir dosya veya klasör mevcut. Lütfen alternatif bir geri yükleme yolu girin.');
      } else {
        alert(describeError(err, 'Geri yükleme başarısız'));
      }
    } finally {
      setActionLoadingId(null);
    }
  };

  const handlePermanentDelete = async () => {
    if (!deleteConfirmItem) return;

    try {
      setActionLoadingId(deleteConfirmItem.trash_id);
      await trashApi.delete(deleteConfirmItem.trash_id, '/');
      setDeleteConfirmItem(null);
      await fetchTrash(page);
    } catch (err: unknown) {
      alert(describeError(err, 'Kalıcı silme işlemi başarısız'));
    } finally {
      setActionLoadingId(null);
    }
  };

  const totalPages = Math.ceil(total / perPage) || 1;

  return (
    <div className="space-y-4">
      {/* Header bar */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-charcoal-900">Çöp Kutusu</h2>
          <p className="text-xs text-charcoal-500">
            Silinmiş yerel dosya ve klasörleri görüntüleyin, geri yükleyin veya kalıcı olarak temizleyin.
          </p>
        </div>
        <button
          type="button"
          onClick={() => fetchTrash(page)}
          disabled={loading}
          className="flex items-center space-x-1 px-3 py-1.5 bg-white border border-charcoal-200 hover:bg-warm-100 rounded-lg text-xs font-medium text-charcoal-700 transition-colors"
          title="Yenile"
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

      {loading && items.length === 0 ? (
        <div className="py-20 text-center text-charcoal-500 space-y-2">
          <SpinnerIcon className="w-8 h-8 text-accent animate-spin mx-auto" />
          <span className="text-sm">Çöp kutusu yükleniyor...</span>
        </div>
      ) : items.length === 0 ? (
        <div className="py-16 text-center text-charcoal-400 bg-white border border-charcoal-200 rounded-xl">
          <TrashIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
          <p className="text-sm font-medium text-charcoal-600">Çöp kutusu boş</p>
          <p className="text-xs text-charcoal-400 mt-1">Silinen öğeler burada saklanır.</p>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="overflow-x-auto bg-white border border-charcoal-200 rounded-xl shadow-sm">
            <table className="w-full table-fixed sm:table-auto text-left text-sm">
              <thead className="bg-warm-100 border-b border-charcoal-200 text-xs font-medium text-charcoal-600 uppercase tracking-wider select-none">
                <tr>
                  <th className="py-3 px-4">Orijinal Yol & Ad</th>
                  <th className="py-3 px-4 w-28 hidden sm:table-cell">Boyut</th>
                  <th className="py-3 px-4 w-44 hidden md:table-cell">Silinme Tarihi</th>
                  <th className="py-3 px-1 sm:px-4 w-20 sm:w-52 text-right">
                    <span className="sr-only sm:not-sr-only">İşlemler</span>
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-charcoal-100">
                {items.map((item) => (
                  <tr key={String(item.trash_id)} className="hover:bg-warm-50/70 transition-colors">
                    <td className="py-3 px-4">
                      <div className="flex items-center space-x-2 min-w-0">
                        {item.is_dir ? (
                          <FolderIcon className="w-5 h-5 text-accent flex-shrink-0" />
                        ) : (
                          <FileIcon className="w-5 h-5 text-charcoal-400 flex-shrink-0" />
                        )}
                        <span className="font-mono text-xs text-charcoal-800 truncate" title={item.original_path}>
                          {item.original_path}
                        </span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-500 hidden sm:table-cell">
                      {item.is_dir ? 'Klasör' : formatBytes(item.size)}
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-500 hidden md:table-cell">
                      {formatDate(item.deleted_at)}
                    </td>
                    <td className="py-3 px-1 sm:px-4 text-right">
                      <div className="flex items-center justify-end space-x-1 sm:space-x-2">
                        <button
                          type="button"
                          onClick={() => handleRestore(item)}
                          disabled={actionLoadingId === item.trash_id}
                          className="flex items-center space-x-1 px-2 sm:px-2.5 py-1 text-xs font-medium text-accent hover:bg-accent/10 border border-accent/30 rounded-lg transition-colors disabled:opacity-40"
                          title="Geri Yükle"
                        >
                          <RestoreIcon className="w-3.5 h-3.5" />
                          <span className="hidden sm:inline">Geri Yükle</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => setDeleteConfirmItem(item)}
                          disabled={actionLoadingId === item.trash_id}
                          className="flex items-center space-x-1 px-2 sm:px-2.5 py-1 text-xs font-medium text-red-600 hover:bg-red-50 border border-red-200 rounded-lg transition-colors disabled:opacity-40"
                          title="Kalıcı Olarak Sil"
                        >
                          <TrashIcon className="w-3.5 h-3.5" />
                          <span className="hidden sm:inline">Kalıcı Sil</span>
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {total > perPage && (
            <div className="flex items-center justify-between px-3 py-2 bg-white border border-charcoal-200 rounded-xl text-xs text-charcoal-600">
              <span>
                Toplam {total} silinmiş öğeden {(page - 1) * perPage + 1} -{' '}
                {Math.min(page * perPage, total)} arası gösteriliyor
              </span>
              <div className="flex items-center space-x-2">
                <button
                  type="button"
                  onClick={() => fetchTrash(page - 1)}
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
                  onClick={() => fetchTrash(page + 1)}
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

      {/* 409 Collision Target Path Modal */}
      {collisionItem && (
        <Modal
          isOpen={true}
          onClose={() => setCollisionItem(null)}
          title="Geri Yükleme Çakışması"
          icon={<AlertTriangleIcon className="w-5 h-5 text-yellow-600" />}
        >
          <div className="space-y-4">
            <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-xl text-xs text-yellow-900">
              {collisionError}
            </div>

            <div>
              <label htmlFor="target-restore-path" className="block text-xs font-medium text-charcoal-700 mb-1">
                Alternatif Geri Yükleme Yolu
              </label>
              <input
                id="target-restore-path"
                type="text"
                value={customTargetPath}
                onChange={(e) => setCustomTargetPath(e.target.value)}
                placeholder="/restored/dosya.txt"
                className="w-full px-3 py-2 border border-charcoal-200 rounded-lg text-xs font-mono focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
              />
            </div>

            <div className="flex justify-end space-x-2 pt-2">
              <button
                type="button"
                onClick={() => setCollisionItem(null)}
                className="px-3.5 py-1.5 text-xs font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg"
              >
                İptal
              </button>
              <button
                type="button"
                onClick={() => handleRestore(collisionItem, customTargetPath)}
                disabled={!customTargetPath.trim() || actionLoadingId !== null}
                className="px-4 py-1.5 text-xs font-medium bg-accent hover:bg-accent-hover text-white rounded-lg shadow-sm disabled:opacity-50"
              >
                Bu Konuma Geri Yükle
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Permanent Delete Confirmation Dialog */}
      {deleteConfirmItem && (
        <Modal
          isOpen={true}
          onClose={() => setDeleteConfirmItem(null)}
          title="Kalıcı Olarak Sil"
          icon={<TrashIcon className="w-5 h-5 text-red-600" />}
          maxWidthClass="max-w-sm"
        >
          <div className="space-y-4">
            <p className="text-xs text-charcoal-600 break-all">
              <span className="font-semibold text-charcoal-900">{deleteConfirmItem.original_path}</span> öğesini
              çöp kutusundan kalıcı olarak silmek istediğinize emin misiniz? Bu işlem geri alınamaz.
            </p>

            <div className="flex justify-end space-x-2 pt-2">
              <button
                type="button"
                onClick={() => setDeleteConfirmItem(null)}
                className="px-3.5 py-1.5 text-xs font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg"
              >
                Vazgeç
              </button>
              <button
                type="button"
                onClick={handlePermanentDelete}
                disabled={actionLoadingId !== null}
                className="px-4 py-1.5 text-xs font-medium bg-red-600 hover:bg-red-700 text-white rounded-lg shadow-sm disabled:opacity-50"
              >
                {actionLoadingId !== null ? 'Siliniyor...' : 'Evet, Kalıcı Olarak Sil'}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};
