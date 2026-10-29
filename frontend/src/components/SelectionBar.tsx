/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files (FilesListTableHeaderActions.vue)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

import React from 'react';
import { FileItem } from '../types';
import { DownloadIcon, MoveIcon, TrashIcon, CloseIcon } from './Icons';
import { useTranslation } from '../utils/i18n';

interface SelectionBarProps {
  selectedItems: FileItem[];
  onClearSelection: () => void;
  onBatchDownload: () => void;
  onBatchMove: () => void;
  onBatchDelete: () => void;
}

export const SelectionBar: React.FC<SelectionBarProps> = ({
  selectedItems,
  onClearSelection,
  onBatchDownload,
  onBatchMove,
  onBatchDelete,
}) => {
  const { t } = useTranslation();
  const count = selectedItems.length;

  if (count === 0) return null;

  const hasDownloadableFiles = selectedItems.some((item) => !item.is_dir);

  return (
    <div
      role="region"
      aria-label={t('files.selection_bar_aria')}
      className="flex items-center justify-between px-4 py-2 bg-accent/10 border border-accent/20 rounded-xl text-xs text-charcoal-800 transition-all shadow-xs"
    >
      <div className="flex items-center space-x-3">
        <span className="font-semibold text-accent">
          {t('files.selected_count', { count })}
        </span>
        <button
          type="button"
          onClick={onClearSelection}
          className="text-charcoal-500 hover:text-charcoal-800 hover:underline flex items-center space-x-1"
        >
          <CloseIcon className="w-3.5 h-3.5" />
          <span>{t('common.cancel')}</span>
        </button>
      </div>

      <div className="flex items-center space-x-1.5">
        {hasDownloadableFiles && (
          <button
            type="button"
            onClick={onBatchDownload}
            className="flex items-center space-x-1 px-2.5 py-1.5 bg-white border border-charcoal-200 hover:bg-charcoal-50 text-charcoal-700 rounded-lg font-medium transition-colors"
            title={t('common.download')}
          >
            <DownloadIcon className="w-3.5 h-3.5 text-charcoal-600" />
            <span>{t('common.download')}</span>
          </button>
        )}

        <button
          type="button"
          onClick={onBatchMove}
          className="flex items-center space-x-1 px-2.5 py-1.5 bg-white border border-charcoal-200 hover:bg-charcoal-50 text-charcoal-700 rounded-lg font-medium transition-colors"
          title={t('common.move')}
        >
          <MoveIcon className="w-3.5 h-3.5 text-charcoal-600" />
          <span>{t('common.move')}</span>
        </button>

        <button
          type="button"
          onClick={onBatchDelete}
          className="flex items-center space-x-1 px-2.5 py-1.5 bg-red-600 hover:bg-red-700 text-white rounded-lg font-medium transition-colors shadow-xs"
          title={t('common.delete')}
        >
          <TrashIcon className="w-3.5 h-3.5" />
          <span>{t('common.delete')}</span>
        </button>
      </div>
    </div>
  );
};
