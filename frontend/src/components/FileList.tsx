import React from 'react';
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

interface FileListProps {
  items: FileItem[];
  onOpenFolder: (item: FileItem) => void;
  onPreview: (item: FileItem) => void;
  onEdit?: (item: FileItem) => void;
  onVersions?: (item: FileItem) => void;
  onDownload: (item: FileItem) => void;
  onShare: (item: FileItem) => void;
  onRename: (item: FileItem) => void;
  onMove: (item: FileItem) => void;
  onDelete: (item: FileItem) => void;
}

export const FileList: React.FC<FileListProps> = ({
  items,
  onOpenFolder,
  onPreview,
  onEdit,
  onVersions,
  onDownload,
  onShare,
  onRename,
  onMove,
  onDelete,
}) => {
  // Menu position is computed on open and rendered `fixed` so the table's
  // overflow-x-auto container cannot clip it on small screens.
  const [openMenu, setOpenMenu] = React.useState<{
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
    const menuWidth = 176; // w-44
    const menuHeight = 340;
    const openUp = window.innerHeight - rect.bottom < menuHeight && rect.top > menuHeight;
    const right = Math.min(
      Math.max(8, window.innerWidth - rect.right),
      Math.max(8, window.innerWidth - menuWidth - 8)
    );
    setOpenMenu({
      id,
      right,
      ...(openUp
        ? { bottom: window.innerHeight - rect.top + 4 }
        : { top: rect.bottom + 4 }),
    });
  };

  if (items.length === 0) {
    return (
      <div className="py-16 text-center text-charcoal-400">
        <FolderIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
        <p className="text-sm">Bu klasörde görüntülenecek öğe bulunamadı.</p>
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

  return (
    <div className="overflow-x-auto bg-white border border-charcoal-200 rounded-xl">
      <table className="w-full table-fixed sm:table-auto text-left text-sm">
        <thead className="bg-warm-100 border-b border-charcoal-200 text-xs font-medium text-charcoal-600 uppercase tracking-wider select-none">
          <tr>
            <th className="py-3 px-4">Ad</th>
            <th className="py-3 px-4 w-28 hidden sm:table-cell">Boyut</th>
            <th className="py-3 px-4 w-44 hidden md:table-cell">Değiştirilme</th>
            <th className="py-3 px-1 sm:px-4 w-12 sm:w-44 text-right">
              <span className="sr-only sm:not-sr-only">İşlemler</span>
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-charcoal-100">
          {items.map((item) => {
            const previewEligible = !item.is_dir && getPreviewEligibility(item.name, item.size).canPreview;

            return (
              <tr
                key={item.name}
                className="hover:bg-warm-50/70 transition-colors group"
              >
                <td className="py-2.5 px-4">
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
                    className="flex w-full min-w-0 items-center space-x-2.5 text-left text-charcoal-800 hover:text-accent font-medium"
                    title={item.name}
                  >
                    {renderIcon(item)}
                    <span className="truncate">{item.name}</span>
                  </button>
                </td>
                <td className="py-2.5 px-4 text-charcoal-500 text-xs hidden sm:table-cell">
                  {item.is_dir ? '-' : formatBytes(item.size)}
                </td>
                <td className="py-2.5 px-4 text-charcoal-500 text-xs hidden md:table-cell">
                  {formatDate(item.modified)}
                </td>
                <td className="py-2.5 px-1 sm:px-4 text-right">
                  {/* Desktop actions: visible on sm and up */}
                  <div className="hidden sm:flex items-center justify-end space-x-1 opacity-75 group-hover:opacity-100 transition-opacity">
                    {previewEligible && (
                      <button
                        type="button"
                        onClick={() => onPreview(item)}
                        className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                        title="Önizle"
                      >
                        <EyeIcon className="w-4 h-4" />
                      </button>
                    )}
                    {!item.is_dir && isEditableTextFile(item.name) && onEdit && (
                      <button
                        type="button"
                        onClick={() => onEdit(item)}
                        className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                        title="Metni Düzenle"
                      >
                        <FileTextIcon className="w-4 h-4 text-emerald-600" />
                      </button>
                    )}
                    {!item.is_dir && onVersions && (
                      <button
                        type="button"
                        onClick={() => onVersions(item)}
                        className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                        title="Sürüm Geçmişi"
                      >
                        <HistoryIcon className="w-4 h-4" />
                      </button>
                    )}
                    {!item.is_dir && (
                      <button
                        type="button"
                        onClick={() => onDownload(item)}
                        className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                        title="İndir"
                      >
                        <DownloadIcon className="w-4 h-4" />
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={() => onShare(item)}
                      className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                      title="Paylaş"
                    >
                      <ShareIcon className="w-4 h-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => onRename(item)}
                      className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                      title="Yeniden Adlandır"
                    >
                      <EditIcon className="w-4 h-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => onMove(item)}
                      className="p-1.5 text-charcoal-500 hover:text-charcoal-900 hover:bg-charcoal-100 rounded"
                      title="Taşı"
                    >
                      <MoveIcon className="w-4 h-4" />
                    </button>
                    <button
                      type="button"
                      onClick={() => onDelete(item)}
                      className="p-1.5 text-charcoal-500 hover:text-red-600 hover:bg-red-50 rounded"
                      title="Sil"
                    >
                      <TrashIcon className="w-4 h-4" />
                    </button>
                  </div>

                  {/* Mobile compact overflow menu: visible below sm */}
                  <div className="sm:hidden flex justify-end">
                    <button
                      type="button"
                      onClick={(e) => toggleMenu(e, item.name)}
                      className="p-1.5 text-charcoal-600 hover:text-charcoal-900 rounded-lg hover:bg-charcoal-100 focus:outline-none focus:ring-2 focus:ring-accent/40"
                      aria-label={`${item.name} için işlemler menüsü`}
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
                                <span>Önizle</span>
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
                                <span>Metni Düzenle</span>
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
                                <span>Sürüm Geçmişi</span>
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
                                <span>İndir</span>
                              </button>
                            )}
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
                              <span>Paylaş</span>
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
                              <span>Yeniden Adlandır</span>
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
                              <span>Taşı</span>
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
                              <TrashIcon className="w-3.5 h-3.5 text-red-600" />
                              <span>Sil</span>
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
