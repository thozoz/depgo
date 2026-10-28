import React from 'react';
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
} from './Icons';

interface FileGridProps {
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

export const FileGrid: React.FC<FileGridProps> = ({
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
  if (items.length === 0) {
    return (
      <div className="py-16 text-center text-charcoal-400">
        <FolderIcon className="w-12 h-12 mx-auto mb-2 text-charcoal-300 stroke-1" />
        <p className="text-sm">Bu klasörde görüntülenecek öğe bulunamadı.</p>
      </div>
    );
  }

  const renderCardIcon = (item: FileItem) => {
    if (item.is_dir) {
      return (
        <div className="w-12 h-12 rounded-xl bg-accent/10 text-accent flex items-center justify-center">
          <FolderIcon className="w-7 h-7" />
        </div>
      );
    }
    if (isInlineRasterImage(item.name)) {
      return (
        <div className="w-12 h-12 rounded-xl bg-blue-50 text-blue-500 flex items-center justify-center">
          <ImageIcon className="w-7 h-7" />
        </div>
      );
    }
    if (isPdf(item.name)) {
      return (
        <div className="w-12 h-12 rounded-xl bg-red-50 text-red-500 flex items-center justify-center">
          <PdfIcon className="w-7 h-7" />
        </div>
      );
    }
    return (
      <div className="w-12 h-12 rounded-xl bg-charcoal-100 text-charcoal-500 flex items-center justify-center">
        <FileIcon className="w-7 h-7" />
      </div>
    );
  };

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
      {items.map((item) => {
        const previewEligible = !item.is_dir && getPreviewEligibility(item.name, item.size).canPreview;

        return (
          <div
            key={item.name}
            className="group relative flex flex-col justify-between p-3.5 bg-white border border-charcoal-200 hover:border-charcoal-300 hover:shadow-sm rounded-xl transition-all"
          >
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
              className="flex flex-col items-center text-center w-full focus:outline-none mb-2"
            >
              <div className="mb-2.5 transition-transform group-hover:scale-105">
                {renderCardIcon(item)}
              </div>
              <span
                className="w-full text-xs font-medium text-charcoal-800 group-hover:text-accent truncate px-1"
                title={item.name}
              >
                {item.name}
              </span>
              <span className="text-[11px] text-charcoal-400 mt-0.5">
                {item.is_dir ? 'Klasör' : formatBytes(item.size)}
              </span>
            </button>

            {/* Quick action bar */}
            <div className="flex flex-wrap items-center justify-center gap-1 pt-2 border-t border-charcoal-100 opacity-90 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity">
              {previewEligible && (
                <button
                  type="button"
                  onClick={() => onPreview(item)}
                  className="p-1 text-charcoal-500 hover:text-charcoal-900 rounded hover:bg-charcoal-100"
                  title="Önizle"
                >
                  <EyeIcon className="w-3.5 h-3.5" />
                </button>
              )}
              {!item.is_dir && isEditableTextFile(item.name) && onEdit && (
                <button
                  type="button"
                  onClick={() => onEdit(item)}
                  className="p-1 text-charcoal-500 hover:text-charcoal-900 rounded hover:bg-charcoal-100"
                  title="Metni Düzenle"
                >
                  <FileTextIcon className="w-3.5 h-3.5 text-emerald-600" />
                </button>
              )}
              {!item.is_dir && onVersions && (
                <button
                  type="button"
                  onClick={() => onVersions(item)}
                  className="p-1 text-charcoal-500 hover:text-charcoal-900 rounded hover:bg-charcoal-100"
                  title="Sürüm Geçmişi"
                >
                  <HistoryIcon className="w-3.5 h-3.5" />
                </button>
              )}
              {!item.is_dir && (
                <button
                  type="button"
                  onClick={() => onDownload(item)}
                  className="p-1 text-charcoal-500 hover:text-charcoal-900 rounded hover:bg-charcoal-100"
                  title="İndir"
                >
                  <DownloadIcon className="w-3.5 h-3.5" />
                </button>
              )}
              <button
                type="button"
                onClick={() => onShare(item)}
                className="p-1 text-charcoal-500 hover:text-charcoal-900 rounded hover:bg-charcoal-100"
                title="Paylaş"
              >
                <ShareIcon className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() => onRename(item)}
                className="p-1 text-charcoal-500 hover:text-charcoal-900 rounded hover:bg-charcoal-100"
                title="Yeniden Adlandır"
              >
                <EditIcon className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() => onMove(item)}
                className="p-1 text-charcoal-500 hover:text-charcoal-900 rounded hover:bg-charcoal-100"
                title="Taşı"
              >
                <MoveIcon className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() => onDelete(item)}
                className="p-1 text-charcoal-500 hover:text-red-600 rounded hover:bg-red-50"
                title="Sil"
              >
                <TrashIcon className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        );
      })}
    </div>
  );
};
