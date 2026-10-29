/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files (FileEntryActions.vue)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

import React, { useEffect, useRef } from 'react';
import { FileItem } from '../types';
import {
  isEditableTextFile,
  getPreviewEligibility,
} from '../utils/format';
import {
  EyeIcon,
  FileTextIcon,
  HistoryIcon,
  DownloadIcon,
  ShareIcon,
  EditIcon,
  MoveIcon,
  TrashIcon,
} from './Icons';
import { useTranslation } from '../utils/i18n';

export interface ContextMenuPosition {
  x: number;
  y: number;
  item: FileItem;
}

interface ContextMenuProps {
  position: ContextMenuPosition | null;
  onClose: () => void;
  onPreview: (item: FileItem) => void;
  onEdit?: (item: FileItem) => void;
  onVersions?: (item: FileItem) => void;
  onDownload: (item: FileItem) => void;
  onShare: (item: FileItem) => void;
  onRename: (item: FileItem) => void;
  onMove: (item: FileItem) => void;
  onDelete: (item: FileItem) => void;
}

export const ContextMenu: React.FC<ContextMenuProps> = ({
  position,
  onClose,
  onPreview,
  onEdit,
  onVersions,
  onDownload,
  onShare,
  onRename,
  onMove,
  onDelete,
}) => {
  const { t } = useTranslation();
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    if (position) {
      window.addEventListener('keydown', handleKeyDown);
    }
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [position, onClose]);

  if (!position) return null;

  const { item, x, y } = position;
  const previewEligible = !item.is_dir && getPreviewEligibility(item.name, item.size).canPreview;

  // Screen boundary detection:
  const menuWidth = 180;
  const menuHeight = 320;
  const left = Math.min(Math.max(8, x), Math.max(8, window.innerWidth - menuWidth - 8));
  const top = y + menuHeight > window.innerHeight && y > menuHeight
    ? Math.max(8, y - menuHeight)
    : Math.min(y, Math.max(8, window.innerHeight - menuHeight - 8));

  return (
    <>
      <div
        className="fixed inset-0 z-40"
        onClick={onClose}
        onContextMenu={(e) => {
          e.preventDefault();
          onClose();
        }}
        aria-hidden="true"
      />
      <div
        ref={menuRef}
        role="menu"
        aria-label={`${item.name} işlemleri`}
        style={{ left, top }}
        className="fixed z-50 w-44 max-h-[75vh] overflow-y-auto bg-white border border-charcoal-200 rounded-xl shadow-2xl py-1 text-left text-xs divide-y divide-charcoal-100 animate-in fade-in zoom-in-95 duration-100"
      >
        <div className="py-1">
          {previewEligible && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onClose();
                onPreview(item);
              }}
              className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100 transition-colors"
            >
              <EyeIcon className="w-3.5 h-3.5 text-charcoal-500" />
              <span>{t('common.preview')}</span>
            </button>
          )}
          {!item.is_dir && isEditableTextFile(item.name) && onEdit && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onClose();
                onEdit(item);
              }}
              className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100 transition-colors"
            >
              <FileTextIcon className="w-3.5 h-3.5 text-emerald-600" />
              <span>{t('common.edit')}</span>
            </button>
          )}
          {!item.is_dir && onVersions && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onClose();
                onVersions(item);
              }}
              className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100 transition-colors"
            >
              <HistoryIcon className="w-3.5 h-3.5 text-charcoal-500" />
              <span>{t('common.versions')}</span>
            </button>
          )}
          {!item.is_dir && (
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                onClose();
                onDownload(item);
              }}
              className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100 transition-colors"
            >
              <DownloadIcon className="w-3.5 h-3.5 text-charcoal-500" />
              <span>{t('common.download')}</span>
            </button>
          )}
        </div>

        <div className="py-1">
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onClose();
              onShare(item);
            }}
            className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100 transition-colors"
          >
            <ShareIcon className="w-3.5 h-3.5 text-charcoal-500" />
            <span>{t('common.share')}</span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onClose();
              onRename(item);
            }}
            className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100 transition-colors"
          >
            <EditIcon className="w-3.5 h-3.5 text-charcoal-500" />
            <span>{t('common.rename')}</span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onClose();
              onMove(item);
            }}
            className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100 transition-colors"
          >
            <MoveIcon className="w-3.5 h-3.5 text-charcoal-500" />
            <span>{t('common.move')}</span>
          </button>
        </div>

        <div className="py-1">
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              onClose();
              onDelete(item);
            }}
            className="w-full flex items-center space-x-2 px-3 py-2 text-red-600 hover:bg-red-50 transition-colors"
          >
            <TrashIcon className="w-3.5 h-3.5 text-red-500" />
            <span>{t('common.delete')}</span>
          </button>
        </div>
      </div>
    </>
  );
};
