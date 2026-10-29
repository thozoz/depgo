/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files Trashbin
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

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
import { useTranslation } from '../utils/i18n';

export const TrashManagement: React.FC = () => {
  const { t } = useTranslation();
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
        setError(describeError(err, t('trash.load_failed')));
      } finally {
        setLoading(false);
      }
    },
    [t]
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
        setCollisionItem(item);
        setCustomTargetPath(item.original_path + '.restored');
        setCollisionError(t('trash.collision_error'));
      } else {
        alert(describeError(err, t('trash.restore_failed')));
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
      alert(describeError(err, t('trash.delete_failed')));
    } finally {
      setActionLoadingId(null);
    }
  };

  const totalPages = Math.ceil(total / perPage) || 1;

  return (
    <div className="space-y-4">
      {/* Header bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-white border border-charcoal-200 rounded-xl px-4 py-3 shadow-xs">
        <div>
          <h2 className="text-base font-semibold text-charcoal-900">{t('trash.title')}</h2>
          <p className="text-xs text-charcoal-500">
            {t('trash.subtitle')}
          </p>
        </div>
        <button
          type="button"
          onClick={() => fetchTrash(page)}
          disabled={loading}
          className="flex items-center space-x-1.5 px-3 py-1.5 bg-warm-50 hover:bg-warm-100 border border-charcoal-200 rounded-lg text-xs font-semibold text-charcoal-700 transition-colors self-start sm:self-auto"
          title={t('common.refresh')}
        >
          <RefreshIcon className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          <span>{t('common.refresh')}</span>
        </button>
      </div>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-xs rounded-xl">
          {error}
        </div>
      )}

      {loading && items.length === 0 ? (
        <div className="py-20 text-center text-charcoal-500 bg-white border border-charcoal-200 rounded-xl">
          <SpinnerIcon className="w-8 h-8 text-accent animate-spin mx-auto mb-2" />
          <span className="text-sm">{t('trash.loading')}</span>
        </div>
      ) : items.length === 0 ? (
        <div className="py-20 text-center text-charcoal-400 bg-white border border-charcoal-200 rounded-xl">
          <TrashIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
          <p className="text-sm font-semibold text-charcoal-700">{t('trash.empty')}</p>
          <p className="text-xs text-charcoal-500 mt-1">{t('trash.empty_desc')}</p>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="overflow-x-auto bg-white border border-charcoal-200 rounded-xl shadow-xs">
            <table className="w-full table-fixed sm:table-auto text-left text-sm border-collapse">
              <thead className="bg-warm-50/80 border-b border-charcoal-200 text-xs font-semibold text-charcoal-600 uppercase tracking-wider select-none">
                <tr>
                  <th className="py-3 px-4">{t('trash.original_path')}</th>
                  <th className="py-3 px-4 w-28 hidden sm:table-cell">{t('files.size')}</th>
                  <th className="py-3 px-4 w-44 hidden md:table-cell">{t('trash.deleted_at')}</th>
                  <th className="py-3 px-2 sm:px-4 w-24 sm:w-56 text-right">
                    <span className="sr-only sm:not-sr-only">{t('common.actions')}</span>
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
                        <span className="font-mono text-xs text-charcoal-900 truncate" title={item.original_path}>
                          {item.original_path}
                        </span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-500 hidden sm:table-cell">
                      {item.is_dir ? t('files.folder') : formatBytes(item.size)}
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-500 hidden md:table-cell">
                      {formatDate(item.deleted_at)}
                    </td>
                    <td className="py-3 px-2 sm:px-4 text-right">
                      <div className="flex items-center justify-end space-x-1.5">
                        <button
                          type="button"
                          onClick={() => handleRestore(item)}
                          disabled={actionLoadingId === item.trash_id}
                          className="flex items-center space-x-1 px-2.5 py-1 text-xs font-semibold text-accent hover:bg-accent/10 border border-accent/30 rounded-lg transition-colors disabled:opacity-40"
                          title={t('trash.restore')}
                        >
                          <RestoreIcon className="w-3.5 h-3.5" />
                          <span className="hidden sm:inline">{t('trash.restore')}</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => setDeleteConfirmItem(item)}
                          disabled={actionLoadingId === item.trash_id}
                          className="flex items-center space-x-1 px-2.5 py-1 text-xs font-semibold text-red-600 hover:bg-red-50 border border-red-200 rounded-lg transition-colors disabled:opacity-40"
                          title={t('trash.permanent_delete')}
                        >
                          <TrashIcon className="w-3.5 h-3.5" />
                          <span className="hidden sm:inline">{t('trash.permanent_delete')}</span>
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
            <div className="flex items-center justify-between px-2 py-1 text-xs text-charcoal-600">
              <span>
                {t('files.pagination_summary', { total, start: (page - 1) * perPage + 1, end: Math.min(page * perPage, total) })}
              </span>
              <div className="flex items-center space-x-2">
                <button
                  type="button"
                  onClick={() => fetchTrash(page - 1)}
                  disabled={page <= 1 || loading}
                  className="px-2.5 py-1 bg-white border border-charcoal-200 rounded-md hover:bg-warm-100 disabled:opacity-40"
                >
                  {t('files.previous')}
                </button>
                <span className="font-semibold text-charcoal-800">
                  {page} / {totalPages}
                </span>
                <button
                  type="button"
                  onClick={() => fetchTrash(page + 1)}
                  disabled={page >= totalPages || loading}
                  className="px-2.5 py-1 bg-white border border-charcoal-200 rounded-md hover:bg-warm-100 disabled:opacity-40"
                >
                  {t('files.next')}
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Collision Modal */}
      {collisionItem && (
        <Modal
          isOpen={true}
          onClose={() => {
            setCollisionItem(null);
            setCustomTargetPath('');
            setCollisionError(null);
          }}
          title={t('trash.collision_title')}
          icon={<AlertTriangleIcon className="w-5 h-5 text-yellow-600" />}
        >
          <div className="space-y-4">
            <p className="text-sm text-charcoal-600">
              {t('trash.collision_desc', { path: collisionItem.original_path })}
            </p>

            <div>
              <label htmlFor="custom-target-path" className="block text-xs font-semibold text-charcoal-700 mb-1">
                {t('trash.custom_path_label')}
              </label>
              <input
                id="custom-target-path"
                type="text"
                value={customTargetPath}
                onChange={(e) => setCustomTargetPath(e.target.value)}
                placeholder="/yeni/konum/dosya.ext"
                className="w-full px-3 py-2 text-sm border border-charcoal-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-accent/40 focus:border-accent"
              />
            </div>

            {collisionError && (
              <div className="p-2.5 bg-yellow-50 border border-yellow-200 text-yellow-800 text-xs rounded-lg">
                {collisionError}
              </div>
            )}

            <div className="flex justify-end space-x-2 pt-2">
              <button
                type="button"
                onClick={() => {
                  setCollisionItem(null);
                  setCustomTargetPath('');
                  setCollisionError(null);
                }}
                className="px-3.5 py-2 text-sm font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                type="button"
                onClick={() => handleRestore(collisionItem, customTargetPath)}
                disabled={!customTargetPath.trim() || actionLoadingId === collisionItem.trash_id}
                className="px-4 py-2 text-sm font-semibold bg-accent hover:bg-accent-hover text-white rounded-lg shadow-xs transition-colors disabled:opacity-50"
              >
                {t('trash.restore_to_path')}
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Permanent Delete Modal */}
      {deleteConfirmItem && (
        <Modal
          isOpen={true}
          onClose={() => setDeleteConfirmItem(null)}
          title={t('trash.permanent_delete_title')}
          icon={<TrashIcon className="w-5 h-5 text-red-600" />}
          maxWidthClass="max-w-sm"
        >
          <div className="space-y-4">
            <p className="text-sm text-charcoal-600 break-words">
              {t('trash.permanent_delete_confirm', { path: deleteConfirmItem.original_path })}
            </p>

            <div className="flex justify-end space-x-2">
              <button
                type="button"
                onClick={() => setDeleteConfirmItem(null)}
                className="px-3.5 py-2 text-sm font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg transition-colors"
              >
                {t('common.cancel')}
              </button>
              <button
                type="button"
                onClick={handlePermanentDelete}
                disabled={actionLoadingId === deleteConfirmItem.trash_id}
                className="px-4 py-2 text-sm font-semibold bg-red-600 hover:bg-red-700 text-white rounded-lg shadow-xs transition-colors disabled:opacity-50"
              >
                {t('common.yes_permanent_delete')}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};
