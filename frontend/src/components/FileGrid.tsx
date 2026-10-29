/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files (FileEntryGrid.vue)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

import React, { useState } from 'react';
import { FileItem } from '../types';
import {
  formatBytes,
  isInlineRasterImage,
  isPdf,
  isEditableTextFile,
  getPreviewEligibility,
} from '../utils/format';
import {
  FolderIcon,
  FileIcon,
  ImageIcon,
  PdfIcon,
  DownloadIcon,
  ShareIcon,
  EditIcon,
  MoveIcon,
  TrashIcon,
  EyeIcon,
  FileTextIcon,
  HistoryIcon,
  MoreVerticalIcon,
} from './Icons';
import { useTranslation } from '../utils/i18n';

interface FileGridProps {
  items: FileItem[];
  selectedNames?: Set<string>;
  onToggleSelect?: (name: string) => void;
  onOpenFolder: (item: FileItem) => void;
  onPreview: (item: FileItem) => void;
  onEdit?: (item: FileItem) => void;
  onVersions?: (item: FileItem) => void;
  onDownload: (item: FileItem) => void;
  onShare: (item: FileItem) => void;
  onRename: (item: FileItem) => void;
  onMove: (item: FileItem) => void;
  onDelete: (item: FileItem) => void;
  onDragStartItem?: (item: FileItem, e: React.DragEvent) => void;
  onDragOverFolder?: (folder: FileItem, e: React.DragEvent) => void;
  onDragLeaveFolder?: (folder: FileItem, e: React.DragEvent) => void;
  onDropToFolder?: (folder: FileItem, e: React.DragEvent) => void;
  dragOverItemName?: string | null;
  onContextMenu?: (item: FileItem, e: React.MouseEvent) => void;
}

export const FileGrid: React.FC<FileGridProps> = ({
  items,
  selectedNames = new Set(),
  onToggleSelect,
  onOpenFolder,
  onPreview,
  onEdit,
  onVersions,
  onDownload,
  onShare,
  onRename,
  onMove,
  onDelete,
  onDragStartItem,
  onDragOverFolder,
  onDragLeaveFolder,
  onDropToFolder,
  dragOverItemName,
  onContextMenu,
}) => {
  const { t } = useTranslation();
  const [activeMenuId, setActiveMenuId] = useState<string | null>(null);

  if (items.length === 0) {
    return (
      <div className="py-20 text-center text-charcoal-400 bg-white border border-charcoal-200 rounded-xl">
        <FolderIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
        <p className="text-sm font-medium">{t('files.empty_directory')}</p>
      </div>
    );
  }

  const renderCardIcon = (item: FileItem) => {
    if (item.is_dir) {
      return (
        <div className="w-14 h-14 rounded-xl bg-accent/10 text-accent flex items-center justify-center">
          <FolderIcon className="w-8 h-8" />
        </div>
      );
    }
    if (isInlineRasterImage(item.name)) {
      return (
        <div className="w-14 h-14 rounded-xl bg-blue-50 text-blue-500 flex items-center justify-center">
          <ImageIcon className="w-8 h-8" />
        </div>
      );
    }
    if (isPdf(item.name)) {
      return (
        <div className="w-14 h-14 rounded-xl bg-red-50 text-red-500 flex items-center justify-center">
          <PdfIcon className="w-8 h-8" />
        </div>
      );
    }
    return (
      <div className="w-14 h-14 rounded-xl bg-charcoal-100 text-charcoal-500 flex items-center justify-center">
        <FileIcon className="w-8 h-8" />
      </div>
    );
  };

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
      {items.map((item) => {
        const previewEligible = !item.is_dir && getPreviewEligibility(item.name, item.size).canPreview;
        const isSelected = selectedNames.has(item.name);
        const isDragOver = dragOverItemName === item.name;

        return (
          <div
            key={item.name}
            draggable={!!onDragStartItem}
            onDragStart={(e) => onDragStartItem?.(item, e)}
            onDragOver={(e) => item.is_dir && onDragOverFolder?.(item, e)}
            onDragLeave={(e) => item.is_dir && onDragLeaveFolder?.(item, e)}
            onDrop={(e) => item.is_dir && onDropToFolder?.(item, e)}
            onContextMenu={(e) => {
              if (onContextMenu) {
                e.preventDefault();
                onContextMenu(item, e);
              }
            }}
            className={`group relative flex flex-col justify-between p-3 bg-white border border-charcoal-200 hover:border-charcoal-300 hover:shadow-sm rounded-xl transition-all ${
              isSelected ? 'ring-2 ring-accent bg-accent/5' : ''
            } ${isDragOver ? 'ring-2 ring-accent bg-accent/20' : ''}`}
          >
            {/* Top row: Checkbox and menu button */}
            <div className="flex items-center justify-between mb-2">
              <div className="w-6 h-6 flex items-center justify-center">
                {onToggleSelect && (
                  <input
                    type="checkbox"
                    aria-label={`${item.name} seç`}
                    checked={isSelected}
                    onChange={() => onToggleSelect(item.name)}
                    className="rounded border-charcoal-300 text-accent focus:ring-accent w-4 h-4 cursor-pointer"
                  />
                )}
              </div>

              <div className="relative">
                <button
                  type="button"
                  onClick={() => setActiveMenuId(activeMenuId === item.name ? null : item.name)}
                  className="p-1 text-charcoal-400 hover:text-charcoal-900 rounded-md hover:bg-charcoal-100"
                  aria-label={t('files.actions_menu', { name: item.name })}
                >
                  <MoreVerticalIcon className="w-4 h-4" />
                </button>

                {activeMenuId === item.name && (
                  <>
                    <div
                      className="fixed inset-0 z-30"
                      onClick={() => setActiveMenuId(null)}
                      aria-hidden="true"
                    />
                    <div
                      role="menu"
                      aria-label={`${item.name} işlemleri`}
                      className="absolute right-0 top-full mt-1 z-40 w-44 bg-white border border-charcoal-200 rounded-xl shadow-xl py-1 text-left text-xs divide-y divide-charcoal-100"
                    >
                      <div className="py-1">
                        {previewEligible && (
                          <button
                            type="button"
                            role="menuitem"
                            onClick={() => {
                              setActiveMenuId(null);
                              onPreview(item);
                            }}
                            className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100"
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
                              setActiveMenuId(null);
                              onEdit(item);
                            }}
                            className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100"
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
                              setActiveMenuId(null);
                              onVersions(item);
                            }}
                            className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100"
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
                              setActiveMenuId(null);
                              onDownload(item);
                            }}
                            className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100"
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
                            setActiveMenuId(null);
                            onShare(item);
                          }}
                          className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100"
                        >
                          <ShareIcon className="w-3.5 h-3.5 text-charcoal-500" />
                          <span>{t('common.share')}</span>
                        </button>
                        <button
                          type="button"
                          role="menuitem"
                          onClick={() => {
                            setActiveMenuId(null);
                            onRename(item);
                          }}
                          className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100"
                        >
                          <EditIcon className="w-3.5 h-3.5 text-charcoal-500" />
                          <span>{t('common.rename')}</span>
                        </button>
                        <button
                          type="button"
                          role="menuitem"
                          onClick={() => {
                            setActiveMenuId(null);
                            onMove(item);
                          }}
                          className="w-full flex items-center space-x-2 px-3 py-2 text-charcoal-700 hover:bg-warm-100"
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
                            setActiveMenuId(null);
                            onDelete(item);
                          }}
                          className="w-full flex items-center space-x-2 px-3 py-2 text-red-600 hover:bg-red-50"
                        >
                          <TrashIcon className="w-3.5 h-3.5 text-red-500" />
                          <span>{t('common.delete')}</span>
                        </button>
                      </div>
                    </div>
                  </>
                )}
              </div>
            </div>

            {/* Middle: Icon / Thumbnail */}
            <button
              type="button"
              onClick={() => {
                if (item.is_dir) {
                  onOpenFolder(item);
                } else if (previewEligible) {
                  onPreview(item);
                } else {
                  onDownload(item);
                }
              }}
              className="flex flex-col items-center justify-center py-3 text-center focus:outline-none"
              title={item.name}
            >
              {renderCardIcon(item)}
              <span className="mt-2 text-xs font-medium text-charcoal-800 line-clamp-2 break-all text-center group-hover:text-accent">
                {item.name}
              </span>
            </button>

            {/* Bottom: Size info */}
            <div className="mt-1 text-center text-[11px] text-charcoal-400">
              {item.is_dir ? '—' : formatBytes(item.size)}
            </div>
          </div>
        );
      })}
    </div>
  );
};
