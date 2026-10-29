/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files (DragAndDropPreview.vue)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

import { forwardRef } from 'react';
import { FolderIcon, FileIcon } from './Icons';
import { t } from '../utils/i18n';

interface DragPreviewProps {
  names: string[];
  isFolder: boolean;
}

export const DragPreview = forwardRef<HTMLDivElement, DragPreviewProps>(
  ({ names, isFolder }, ref) => {
    const isSingle = names.length === 1;
    const count = names.length;

    return (
      <div
        ref={ref}
        style={{ position: 'fixed', top: -9999, left: -9999 }}
        className="pointer-events-none z-50 flex items-center space-x-2 px-3 py-1.5 bg-white border border-charcoal-200 rounded-lg shadow-lg text-xs font-semibold text-charcoal-800"
      >
        <div className="w-5 h-5 flex items-center justify-center flex-shrink-0">
          {isSingle && isFolder ? (
            <FolderIcon className="w-4 h-4 text-accent" />
          ) : (
            <FileIcon className="w-4 h-4 text-charcoal-600" />
          )}
        </div>
        <span className="truncate max-w-[180px]">
          {isSingle ? names[0] : t('files.item_count', { count })}
        </span>
      </div>
    );
  }
);

DragPreview.displayName = 'DragPreview';
