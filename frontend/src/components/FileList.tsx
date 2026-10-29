/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files (FileEntry.vue, FilesListTableHeader.vue)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

import React, { useState } from 'react';
import { FileItem } from '../types';
import {
  formatBytes,
  formatDate,
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

interface FileListProps {
  items: FileItem[];
  selectedNames?: Set<string>;
  onToggleSelect?: (name: string) => void;
  onToggleSelectAll?: () => void;
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

export const FileList: React.FC<FileListProps> = ({
  items,
  selectedNames = new Set(),
  onToggleSelect,
  onToggleSelectAll,
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
  const [openMenu, setOpenMenu] = useState<{
    id: string;
    top?: number;
    bottom?: number;
    right: number;
  } | null>(null);

  const toggleMenu = (e: React.MouseEvent<HTMLButtonElement>, id: string) => {
    if (openMenu?.id === id) {
      setOpenMenu(null);
      return;
    }
    const rect = e.currentTarget.getBoundingClientRect();
    const menuWidth = 176;
    const menuHeight = 340;
    const openUp = window.innerHeight - rect.bottom < menuHeight && rect.top > menuHeight;
    const right = Math.min(
      Math.max(8, window.innerWidth - rect.right),
      Math.max(8, window.innerWidth - menuWidth - 8)
    );
    setOpenMenu({
      id,
      right,
      ...(openUp ? { bottom: window.innerHeight - rect.top + 4 } : { top: rect.bottom + 4 }),
    });
  };

  if (items.length === 0) {
    return (
      <div className="py-20 text-center text-charcoal-400 bg-white border border-charcoal-200 rounded-xl">
        <FolderIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
        <p className="text-sm font-medium">{t('files.empty_directory')}</p>
      </div>
    );
  }

  const renderIcon = (item: FileItem) => {
    if (item.is_dir) {
      return <FolderIcon className="w-5 h-5 text-accent flex-shrink-0" />;
    }
    if (isInlineRasterImage(item.name)) {
      return <ImageIcon className="w-5 h-5 text-blue-500 flex-shrink-0" />;
    }
    if (isPdf(item.name)) {
      return <PdfIcon className="w-5 h-5 text-red-500 flex-shrink-0" />;
    }
    return <FileIcon className="w-5 h-5 text-charcoal-400 flex-shrink-0" />;
  };

  const allSelected = items.length > 0 && items.every((i) => selectedNames.has(i.name));
  const someSelected = items.some((i) => selectedNames.has(i.name));

  return (
    <div className="overflow-x-auto bg-white border border-charcoal-200 rounded-xl shadow-xs">
      <table className="w-full table-fixed sm:table-auto text-left text-sm border-collapse">
        <thead className="bg-warm-50/80 border-b border-charcoal-200 text-xs font-semibold text-charcoal-600 uppercase tracking-wider select-none">
          <tr>
            {/* Checkbox column */}
            <th className="w-10 sm:w-12 py-3 px-3 sm:px-4 text-center">
              {onToggleSelectAll && (
                <input
                  type="checkbox"
                  aria-label={t('files.select_all')}
                  checked={allSelected}
                  ref={(input) => {
                    if (input) input.indeterminate = someSelected && !allSelected;
                  }}
                  onChange={onToggleSelectAll}
                  className="rounded border-charcoal-300 text-accent focus:ring-accent w-4 h-4 cursor-pointer"
                />
              )}
            </th>
            <th className="py-3 px-2 sm:px-4">{t('files.name')}</th>
            <th className="py-3 px-4 w-28 hidden sm:table-cell">{t('files.size')}</th>
            <th className="py-3 px-4 w-44 hidden md:table-cell">{t('files.modified')}</th>
            <th className="py-3 px-2 sm:px-4 w-12 sm:w-56 text-right">
              <span className="sr-only sm:not-sr-only">{t('common.actions')}</span>
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-charcoal-100">
          {items.map((item) => {
            const previewEligible = !item.is_dir && getPreviewEligibility(item.name, item.size).canPreview;
            const isSelected = selectedNames.has(item.name);
            const isDragOver = dragOverItemName === item.name;

            return (
              <tr
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
                className={`transition-colors group ${
                  isSelected ? 'bg-accent/5' : 'hover:bg-warm-50/80'
                } ${isDragOver ? 'ring-2 ring-accent bg-accent/20' : ''}`}
              >
                {/* Row Checkbox */}
                <td className="w-10 sm:w-12 py-2.5 px-3 sm:px-4 text-center">
                  {onToggleSelect && (
                    <input
                      type="checkbox"
                      aria-label={`${item.name} seç`}
                      checked={isSelected}
                      onChange={() => onToggleSelect(item.name)}
                      className="rounded border-charcoal-300 text-accent focus:ring-accent w-4 h-4 cursor-pointer"
                    />
                  )}
                </td>

                {/* Name / Icon */}
                <td className="py-2.5 px-2 sm:px-4">
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
                    className="flex w-full min-w-0 items-center space-x-2.5 text-left text-charcoal-900 hover:text-accent font-medium"
                    title={item.name}
                  >
                    {renderIcon(item)}
                    <span className="truncate">{item.name}</span>
                  </button>
                </td>

                {/* Size */}
                <td className="py-2.5 px-4 text-charcoal-500 text-xs hidden sm:table-cell">
                  {item.is_dir ? '—' : formatBytes(item.size)}
                </td>

                {/* Modified */}
                <td className="py-2.5 px-4 text-charcoal-500 text-xs hidden md:table-cell">
                  {formatDate(item.modified)}
                </td>

                {/* Actions */}
                <td className="py-2.5 px-2 sm:px-4 text-right">
                  {/* Desktop Actions */}
                  <div className="hidden sm:flex items-center justify-end space-x-1 opacity-70 group-hover:opacity-100 transition-opacity">
                    {previewEligible && (
                      <button
                        type="button"
                        onClick={() => onPreview(item)}
                        className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-md transition-colors"
                        title={t('common.preview')}
                        aria-label={t('common.preview')}
                      >
                        <EyeIcon className="w-4 h-4" />
                      </button>
                    )}
                    {!item.is_dir && isEditableTextFile(item.name) && onEdit && (
                      <button
                        type="button"
                        onClick={() => onEdit(item)}
                        className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-md transition-colors"
                        title={t('common.edit')}
                        aria-label={t('common.edit')}
                      >
                        <FileTextIcon className="w-4 h-4 text-emerald-600" />
                      </button>
                    )}
                    {!item.is_dir && onVersions && (
                      <button
                        type="button"
                        onClick={() => onVersions(item)}
                        className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-md transition-colors"
                        title={t('common.versions')}
                        aria-label={t('common.versions')}
                      >
                        <HistoryIcon className="w-4 h-4" />
                      </button>
                    )}
                    {!item.is_dir && (
                      <button
                        type="button"
                        onClick={() => onDownload(item)}
                        className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-md transition-colors"
                        title={t('common.download')}
                        aria-label={t('common.download')}
                      >
                        <DownloadIcon className="w-4 h-4" />
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={() => onShare(item)}
                      className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-md transition-colors"
                      title={t('common.share')}
                      aria-label={t('common.share')}
                    >
                      <ShareIcon className="w-4 h-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => onRename(item)}
                      className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-md transition-colors"
                      title={t('common.rename')}
                      aria-label={t('common.rename')}
                    >
                      <EditIcon className="w-4 h-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => onMove(item)}
                      className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-md transition-colors"
                      title={t('common.move')}
                      aria-label={t('common.move')}
                    >
                      <MoveIcon className="w-4 h-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => onDelete(item)}
                      className="p-1.5 text-charcoal-500 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors"
                      title={t('common.delete')}
                      aria-label={t('common.delete')}
                    >
                      <TrashIcon className="w-4 h-4" />
                    </button>
                  </div>

                  {/* Mobile Actions Menu */}
                  <div className="sm:hidden flex justify-end">
                    <button
                      type="button"
                      onClick={(e) => toggleMenu(e, item.name)}
                      className="p-1.5 text-charcoal-600 hover:text-charcoal-900 rounded-lg hover:bg-charcoal-100 focus:outline-none focus:ring-2 focus:ring-accent/40"
                      aria-label={t('files.actions_menu', { name: item.name })}
                      aria-expanded={openMenu?.id === item.name}
                    >
                      <MoreVerticalIcon className="w-4 h-4" />
                    </button>

                    {openMenu?.id === item.name && (
                      <>
                        <div
                          className="fixed inset-0 z-30"
                          onClick={() => setOpenMenu(null)}
                          aria-hidden="true"
                        />
                        <div
                          role="menu"
                          aria-label={`${item.name} işlemleri`}
                          style={{ right: openMenu.right, top: openMenu.top, bottom: openMenu.bottom }}
                          className="fixed z-40 w-44 max-h-[70vh] overflow-y-auto bg-white border border-charcoal-200 rounded-xl shadow-xl py-1 text-left text-xs divide-y divide-charcoal-100"
                        >
                          <div className="py-1">
                            {previewEligible && (
                              <button
                                type="button"
                                role="menuitem"
                                onClick={() => {
                                  setOpenMenu(null);
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
                                  setOpenMenu(null);
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
                                  setOpenMenu(null);
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
                                  setOpenMenu(null);
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
                                setOpenMenu(null);
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
                                setOpenMenu(null);
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
                                setOpenMenu(null);
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
                                setOpenMenu(null);
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
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};
