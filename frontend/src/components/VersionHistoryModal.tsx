import React, { useState, useEffect, useCallback } from 'react';
import { FileVersionItem } from '../types';
import { versionApi, ApiError } from '../api/client';
import { formatBytes, formatDate } from '../utils/format';
import { Modal } from './Modal';
import { HistoryIcon, RestoreIcon, SpinnerIcon, RefreshIcon, AlertTriangleIcon } from './Icons';
import { describeError } from '../utils/errors';

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
        `Revizyon #${item.revision} sürümüne dönmek istediğinize emin misiniz? Mevcut içerik geçmişe kaydedilerek korunacaktır.`
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
        setError(describeError(err, 'Sürüme geri dönülemedi'));
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
      maxWidthClass="max-w-xl"
    >
      <div className="space-y-3">
        <div className="flex items-center justify-between text-xs text-charcoal-600 pb-2 border-b border-charcoal-100">
          <span className="truncate max-w-xs font-medium text-charcoal-800" title={fileName}>
            {fileName}
          </span>
          <div className="flex items-center space-x-2">
            <span className="font-mono text-charcoal-500">Mevcut Rev: #{currentRevision}</span>
            <button
              type="button"
              onClick={() => fetchVersions(page)}
              disabled={loading}
              className="p-1 text-charcoal-500 hover:text-charcoal-800 rounded"
              title="Yenile"
            >
              <RefreshIcon className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>

        {conflict && (
          <div className="p-3 bg-yellow-50 border border-yellow-200 text-yellow-900 text-xs rounded-lg flex items-center justify-between gap-2">
            <div className="flex items-center space-x-2">
              <AlertTriangleIcon className="w-4 h-4 text-yellow-600 flex-shrink-0" />
              <span>{conflict}</span>
            </div>
            <button
              type="button"
              onClick={() => fetchVersions(1)}
              className="px-2 py-0.5 bg-yellow-600 hover:bg-yellow-700 text-white rounded font-medium text-[11px] whitespace-nowrap"
            >
              Yenile
            </button>
          </div>
        )}

        {error && (
          <div className="p-2.5 bg-red-50 border border-red-200 text-red-700 text-xs rounded-lg">
            {error}
          </div>
        )}

        {loading && versions.length === 0 ? (
          <div className="py-12 text-center text-charcoal-500 space-y-2">
            <SpinnerIcon className="w-6 h-6 text-accent animate-spin mx-auto" />
            <p className="text-xs">Sürüm geçmişi yükleniyor...</p>
          </div>
        ) : versions.length === 0 ? (
          <div className="py-10 text-center text-charcoal-400">
            <HistoryIcon className="w-10 h-10 mx-auto mb-2 text-charcoal-300 stroke-1" />
            <p className="text-sm">Bu dosya için kayıtlı önceki bir sürüm bulunamadı.</p>
          </div>
        ) : (
          <div className="space-y-2 max-h-72 overflow-y-auto pr-1">
            {versions.map((ver) => {
              const isCurrent = ver.revision === currentRevision;

              return (
                <div
                  key={String(ver.version_id)}
                  className={`p-3 rounded-xl border transition-colors flex items-center justify-between ${
                    isCurrent
                      ? 'bg-accent/5 border-accent/30'
                      : 'bg-warm-50 border-charcoal-200 hover:border-charcoal-300'
                  }`}
                >
                  <div className="flex flex-col space-y-0.5">
                    <div className="flex items-center space-x-2">
                      <span className="font-semibold text-xs text-charcoal-900 font-mono">
                        Rev #{ver.revision}
                      </span>
                      {isCurrent && (
                        <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-accent/20 text-accent-dark font-medium">
                          Mevcut
                        </span>
                      )}
                    </div>
                    <div className="text-[11px] text-charcoal-500 space-x-2">
                      <span>{formatDate(ver.created_at)}</span>
                      <span>•</span>
                      <span>{formatBytes(ver.size)}</span>
                    </div>
                  </div>

                  {!isCurrent && (
                    <button
                      type="button"
                      onClick={() => handleRestore(ver)}
                      disabled={restoringId !== null}
                      className="flex items-center space-x-1 px-2.5 py-1.5 text-xs font-medium text-charcoal-700 bg-white border border-charcoal-200 hover:bg-warm-100 rounded-lg transition-colors disabled:opacity-40"
                      title="Bu sürüme geri dön"
                    >
                      {restoringId === ver.version_id ? (
                        <SpinnerIcon className="w-3.5 h-3.5 animate-spin text-accent" />
                      ) : (
                        <RestoreIcon className="w-3.5 h-3.5 text-accent" />
                      )}
                      <span>Geri Dön</span>
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {total > perPage && (
          <div className="flex items-center justify-between pt-2 border-t border-charcoal-100 text-xs text-charcoal-600">
            <span>
              {total} sürümden {(page - 1) * perPage + 1} - {Math.min(page * perPage, total)}
            </span>
            <div className="flex items-center space-x-1.5">
              <button
                type="button"
                onClick={() => fetchVersions(page - 1)}
                disabled={page <= 1 || loading}
                className="px-2 py-0.5 border border-charcoal-200 rounded disabled:opacity-40"
              >
                Önceki
              </button>
              <span>
                {page} / {totalPages}
              </span>
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
      </div>
    </Modal>
  );
};
