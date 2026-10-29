/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files Sharing
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

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
import { describeShareError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

export const ShareManagement: React.FC = () => {
  const { t } = useTranslation();
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
      setError(describeShareError(err, t('share.load_failed')));
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
    if (!window.confirm(t('share.disable_confirm', { name: share.name }))) {
      return;
    }
    try {
      await shareApi.disable(share.share_id);
      fetchShares(page);
    } catch (err: unknown) {
      alert(describeShareError(err, t('share.disable_failed')));
    }
  };

  const handleEnable = async (share: ShareResp) => {
    const spent = share.access_limit > 0 && share.remaining_accesses === 0;
    if (spent && !window.confirm(t('share.enable_spent_confirm'))) {
      return;
    }
    try {
      await shareApi.enable(share.share_id);
      fetchShares(page);
    } catch (err: unknown) {
      alert(describeShareError(err, t('share.enable_failed')));
    }
  };

  const handleDelete = async (shareId: string) => {
    if (!window.confirm(t('share.delete_confirm'))) {
      return;
    }
    try {
      await shareApi.delete(shareId);
      fetchShares(page);
    } catch (err: unknown) {
      alert(describeShareError(err, t('share.delete_failed')));
    }
  };

  const totalPages = Math.ceil(total / perPage) || 1;

  return (
    <div className="space-y-4">
      {/* Header bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-white border border-charcoal-200 rounded-xl px-4 py-3 shadow-xs">
        <div>
          <h2 className="text-base font-semibold text-charcoal-900">{t('share.header_title')}</h2>
          <p className="text-xs text-charcoal-500">
            {t('share.header_desc')}
          </p>
        </div>
        <button
          type="button"
          onClick={() => fetchShares(page)}
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

      {loading && shares.length === 0 ? (
        <div className="py-20 text-center text-charcoal-500 bg-white border border-charcoal-200 rounded-xl">
          <SpinnerIcon className="w-8 h-8 text-accent animate-spin mx-auto mb-2" />
          <span className="text-sm">{t('share.loading_shares')}</span>
        </div>
      ) : shares.length === 0 ? (
        <div className="py-20 text-center text-charcoal-400 bg-white border border-charcoal-200 rounded-xl">
          <ShareIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
          <p className="text-sm font-semibold text-charcoal-700">{t('share.no_active')}</p>
          <p className="text-xs text-charcoal-500 mt-1">
            {t('share.no_active_desc')}
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="overflow-x-auto bg-white border border-charcoal-200 rounded-xl shadow-xs">
            <table className="w-full text-left text-sm border-collapse">
              <thead className="bg-warm-50/80 border-b border-charcoal-200 text-xs font-semibold text-charcoal-600 uppercase tracking-wider select-none">
                <tr>
                  <th className="py-3 px-4">{t('share.th_link_name')}</th>
                  <th className="py-3 px-4 hidden md:table-cell">{t('share.th_root_path')}</th>
                  <th className="py-3 px-4 hidden sm:table-cell">{t('share.th_security')}</th>
                  <th className="py-3 px-4 hidden lg:table-cell">{t('share.th_expiration')}</th>
                  <th className="py-3 px-4 hidden sm:table-cell">{t('share.th_views')}</th>
                  <th className="py-3 px-4">{t('share.th_status')}</th>
                  <th className="py-3 px-4 text-right">{t('share.th_actions')}</th>
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
                            className="p-1 text-charcoal-400 hover:text-charcoal-700 rounded-md transition-colors"
                            title={t('share.copy_button')}
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
                        <span className="inline-flex items-center space-x-1 px-2 py-0.5 rounded-md bg-yellow-50 text-yellow-700 border border-yellow-200">
                          <LockIcon className="w-3 h-3" />
                          <span>{t('share.badge_password')}</span>
                        </span>
                      ) : (
                        <span className="text-charcoal-400">{t('share.badge_public')}</span>
                      )}
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-500 hidden lg:table-cell">
                      {share.expires_at ? formatDate(share.expires_at) : t('share.exp_unlimited')}
                    </td>
                    <td className="py-3 px-4 text-xs text-charcoal-600 hidden sm:table-cell">
                      {t('share.view_count_badge', { count: share.view_count })}
                    </td>
                    <td className="py-3 px-4">
                      <span
                        className={`inline-block whitespace-nowrap px-2.5 py-0.5 rounded-full text-xs font-semibold ${
                          share.enabled
                            ? 'bg-accent/15 text-accent-dark'
                            : 'bg-charcoal-100 text-charcoal-500'
                        }`}
                      >
                        {share.enabled ? t('share.active') : t('share.disabled')}
                      </span>
                      {share.burn_after_read && (
                        <span
                          className="ml-1.5 inline-block px-2 py-0.5 rounded-full text-xs font-medium bg-yellow-50 text-yellow-700 border border-yellow-200"
                          title={t('share.burn_description')}
                        >
                          {t('share.burn_badge')}
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-4 text-right">
                      <div className="flex items-center justify-end space-x-1.5">
                        <button
                          type="button"
                          onClick={() => setEditingShare(share)}
                          className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-md transition-colors"
                          title={t('common.edit')}
                        >
                          <EditIcon className="w-4 h-4" />
                        </button>
                        {share.enabled ? (
                          <button
                            type="button"
                            onClick={() => handleDisable(share)}
                            className="text-xs px-2.5 py-1 rounded-md border border-charcoal-200 text-charcoal-600 hover:bg-charcoal-100 font-medium transition-colors"
                            title={t('share.disable_title')}
                          >
                            {t('share.disable')}
                          </button>
                        ) : (
                          <button
                            type="button"
                            onClick={() => handleEnable(share)}
                            className="text-xs px-2.5 py-1 rounded-md border border-accent/40 text-accent font-semibold hover:bg-accent/10 transition-colors"
                            title={t('share.enable_title')}
                          >
                            {t('share.enable')}
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => handleDelete(share.share_id)}
                          className="p-1.5 text-charcoal-500 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                          title={t('common.delete')}
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
                {t('share.pagination_summary', { total, start: (page - 1) * perPage + 1, end: Math.min(page * perPage, total) })}
              </span>
              <div className="flex items-center space-x-2">
                <button
                  type="button"
                  onClick={() => fetchShares(page - 1)}
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
                  onClick={() => fetchShares(page + 1)}
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
