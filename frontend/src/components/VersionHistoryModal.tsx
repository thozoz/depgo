import React, { useState, useEffect, useCallback } from 'react';
import { FileVersionItem } from '../types';
import { versionApi, ApiError } from '../api/client';
import { formatBytes, formatDate } from '../utils/format';
import { Modal } from './Modal';
import { HistoryIcon, RestoreIcon, SpinnerIcon, RefreshIcon, AlertTriangleIcon } from './Icons';
import { describeError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

interface VersionHistoryModalProps {
  isOpen: boolean;
  filePath: string;
  fileName: string;
  onClose: () => void;
  onRestored: () => void;
}

export const VersionHistoryModal: React.FC<VersionHistoryModalProps> = ({
  isOpen,
  filePath,
  fileName,
  onClose,
  onRestored,
}) => {
  const { t } = useTranslation();
  const [versions, setVersions] = useState<FileVersionItem[]>([]);
  const [currentRevision, setCurrentRevision] = useState<number>(0);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const perPage = 20;

  const [loading, setLoading] = useState(true);
  const [restoringId, setRestoringId] = useState<string | number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [conflict, setConflict] = useState<string | null>(null);

  const fetchVersions = useCallback(
    async (targetPage = 1) => {
      try {
        setLoading(true);
        setError(null);
        setConflict(null);
        const data = await versionApi.list(filePath, targetPage, perPage);
        setVersions(data.content || []);
        setCurrentRevision(data.revision);
        setTotal(data.total || 0);
        setPage(targetPage);
      } catch (err: unknown) {
        setError(describeError(err, 'Sürüm geçmişi alınamadı'));
      } finally {
        setLoading(false);
      }
    },
    [filePath]
  );

  useEffect(() => {
    if (isOpen) {
      fetchVersions(1);
    }
  }, [isOpen, fetchVersions]);

  const handleRestore = async (item: FileVersionItem) => {
    if (
      !window.confirm(
        t('versions.restore_confirm', { revision: item.revision })
      )
    ) {
      return;
    }

    try {
      setRestoringId(item.version_id);
      setError(null);
      setConflict(null);
      await versionApi.restore(filePath, item.version_id, currentRevision);
      onRestored();
      await fetchVersions(1);
    } catch (err: unknown) {
      if (err instanceof ApiError && (err.status === 409 || err.status === 412 || err.code === 409 || err.code === 412)) {
        setConflict(
          'Sürüm çakışması (409/412): Dosya başka bir işlem tarafından değiştirilmiş. Lütfen güncel sürüm listesini yenileyip tekrar deneyin.'
        );
      } else {
        setError(describeError(err, 'Sürüm geri yüklenemedi'));
      }
    } finally {
      setRestoringId(null);
    }
  };

  const totalPages = Math.ceil(total / perPage) || 1;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Sürüm Geçmişi"
      icon={<HistoryIcon className="w-5 h-5 text-accent" />}
      maxWidthClass="max-w-2xl"
    >
      <div className="space-y-4">
        {/* Info header */}
        <div className="flex items-center justify-between pb-3 border-b border-charcoal-100 text-xs">
          <div>
            <span className="text-charcoal-500">Dosya: </span>
            <span className="font-semibold text-charcoal-900">{fileName}</span>
            <span className="ml-2 font-mono px-2 py-0.5 rounded bg-accent/10 text-accent font-medium">
              Aktif Rev #{currentRevision}
            </span>
          </div>

          <button
            type="button"
            onClick={() => fetchVersions(page)}
            disabled={loading}
            className="p-1 text-charcoal-500 hover:text-charcoal-900 rounded hover:bg-charcoal-100"
            title={t('common.refresh')}
          >
            <RefreshIcon className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {conflict && (
          <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-xs text-yellow-800 flex items-start space-x-2">
            <AlertTriangleIcon className="w-4 h-4 text-yellow-600 flex-shrink-0 mt-0.5" />
            <span>{conflict}</span>
          </div>
        )}

        {error && (
          <div className="p-2.5 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700">
            {error}
          </div>
        )}

        {/* Versions list */}
        {loading && versions.length === 0 ? (
          <div className="py-12 text-center text-charcoal-500">
            <SpinnerIcon className="w-6 h-6 text-accent animate-spin mx-auto mb-2" />
            <span className="text-xs">Sürümler yükleniyor...</span>
          </div>
        ) : versions.length === 0 ? (
          <div className="py-12 text-center text-charcoal-400">
            <p className="text-xs">Bu dosya için kaydedilmiş önceki bir sürüm bulunamadı.</p>
          </div>
        ) : (
          <div className="overflow-x-auto border border-charcoal-200 rounded-xl">
            <table className="w-full text-left text-xs">
              <thead className="bg-warm-100 text-charcoal-600 border-b border-charcoal-200 font-medium">
                <tr>
                  <th className="py-2.5 px-3">Revizyon</th>
                  <th className="py-2.5 px-3">Boyut</th>
                  <th className="py-2.5 px-3">Kayıt Tarihi</th>
                  <th className="py-2.5 px-3 text-right">İşlem</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-charcoal-100">
                {versions.map((ver) => {
                  const isCurrent = ver.revision === currentRevision;

                  return (
                    <tr
                      key={String(ver.version_id)}
                      className={`hover:bg-warm-50/70 transition-colors ${
                        isCurrent ? 'bg-accent/5 font-medium' : ''
                      }`}
                    >
                      <td className="py-2.5 px-3 font-mono">
                        <span className="px-1.5 py-0.5 rounded bg-charcoal-100 text-charcoal-700">
                          Rev #{ver.revision}
                        </span>
                        {isCurrent && (
                          <span className="ml-2 text-[11px] text-accent font-semibold">(Mevcut)</span>
                        )}
                      </td>
                      <td className="py-2.5 px-3 text-charcoal-500">{formatBytes(ver.size)}</td>
                      <td className="py-2.5 px-3 text-charcoal-500">{formatDate(ver.created_at)}</td>
                      <td className="py-2.5 px-3 text-right">
                        {!isCurrent && (
                          <button
                            type="button"
                            onClick={() => handleRestore(ver)}
                            disabled={restoringId === ver.version_id}
                            className="inline-flex items-center space-x-1 px-2.5 py-1 bg-white border border-charcoal-200 hover:border-accent hover:text-accent rounded text-[11px] font-medium transition-colors disabled:opacity-40"
                          >
                            <RestoreIcon className="w-3 h-3" />
                            <span>Geri Dön</span>
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {total > perPage && (
          <div className="flex items-center justify-between pt-2 text-xs text-charcoal-500">
            <span>Toplam {total} sürüm</span>
            <div className="flex space-x-1">
              <button
                type="button"
                onClick={() => fetchVersions(page - 1)}
                disabled={page <= 1 || loading}
                className="px-2 py-0.5 border border-charcoal-200 rounded disabled:opacity-40"
              >
                Önceki
              </button>
              <button
                type="button"
                onClick={() => fetchVersions(page + 1)}
                disabled={page >= totalPages || loading}
                className="px-2 py-0.5 border border-charcoal-200 rounded disabled:opacity-40"
              >
                Sonraki
              </button>
            </div>
          </div>
        )}

        <div className="flex justify-end pt-3 border-t border-charcoal-100">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg transition-colors"
          >
            {t('common.close')}
          </button>
        </div>
      </div>
    </Modal>
  );
};
