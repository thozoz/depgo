import React, { useState, useRef } from 'react';
import { fsApi, ApiError } from '../api/client';
import { formatBytes, joinPath } from '../utils/format';
import { Modal } from './Modal';
import { UploadIcon, CheckIcon, SpinnerIcon, CloseIcon } from './Icons';
import { describeError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

interface UploadModalProps {
  isOpen: boolean;
  currentDir: string;
  onClose: () => void;
  onCompleted: () => void;
}

interface FileUploadState {
  file: File;
  progress: number;
  loaded: number;
  total: number;
  status: 'pending' | 'uploading' | 'completed' | 'conflict' | 'error';
  error?: string;
  controller?: AbortController;
}

export const UploadModal: React.FC<UploadModalProps> = ({
  isOpen,
  currentDir,
  onClose,
  onCompleted,
}) => {
  const { t } = useTranslation();
  const [uploads, setUploads] = useState<FileUploadState[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFilesSelected = (files: FileList | null) => {
    if (!files || files.length === 0) return;
    const newUploads: FileUploadState[] = Array.from(files).map((file) => ({
      file,
      progress: 0,
      loaded: 0,
      total: file.size,
      status: 'pending',
    }));
    setUploads((prev) => [...prev, ...newUploads]);
  };

  const uploadSingle = async (index: number, overwrite = false): Promise<boolean> => {
    const item = uploads[index];
    if (!item) return false;

    const controller = new AbortController();
    const targetPath = joinPath(currentDir, item.file.name);

    setUploads((prev) =>
      prev.map((u, idx) => (idx === index ? { ...u, status: 'uploading', controller, error: undefined } : u))
    );

    try {
      await fsApi.uploadStream(
        targetPath,
        item.file,
        (percent, loaded, total) => {
          setUploads((prev) =>
            prev.map((u, idx) =>
              idx === index ? { ...u, progress: percent, loaded, total } : u
            )
          );
        },
        controller.signal,
        overwrite
      );

      setUploads((prev) =>
        prev.map((u, idx) => (idx === index ? { ...u, status: 'completed', progress: 100 } : u))
      );
      return true;
    } catch (err: unknown) {
      if (err instanceof ApiError && (err.status === 403 || err.message.toLowerCase().includes('exist'))) {
        setUploads((prev) =>
          prev.map((u, idx) =>
            idx === index
              ? {
                  ...u,
                  status: 'conflict',
                  error: t('upload.conflict_warning'),
                }
              : u
          )
        );
      } else {
        const errorMsg = describeError(err, 'Yükleme başarısız');
        setUploads((prev) =>
          prev.map((u, idx) => (idx === index ? { ...u, status: 'error', error: errorMsg } : u))
        );
      }
      return false;
    }
  };

  const startUploads = async () => {
    if (isUploading) return;
    setIsUploading(true);

    try {
      for (let i = 0; i < uploads.length; i++) {
        if (uploads[i].status === 'pending') {
          await uploadSingle(i);
        }
      }
    } finally {
      setIsUploading(false);
      onCompleted();
    }
  };

  const handleExplicitOverwrite = async (index: number) => {
    await uploadSingle(index, true);
    onCompleted();
  };

  const removePending = (index: number) => {
    setUploads((prev) => prev.filter((_, idx) => idx !== index));
  };

  const allFinished = uploads.length > 0 && uploads.every((u) => u.status === 'completed');

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Dosya Yükle"
      icon={<UploadIcon className="w-5 h-5 text-accent" />}
      maxWidthClass="max-w-md"
    >
      {/* Drag & drop upload area */}
      <div
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragOver(false);
          handleFilesSelected(e.dataTransfer.files);
        }}
        onClick={() => fileInputRef.current?.click()}
        className={`border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-colors mb-4 ${
          dragOver
            ? 'border-accent bg-accent/5'
            : 'border-charcoal-300 hover:border-accent hover:bg-warm-50'
        }`}
      >
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={(e) => handleFilesSelected(e.target.files)}
        />
        <UploadIcon className="w-8 h-8 mx-auto text-charcoal-400 mb-2" />
        <p className="text-xs font-semibold text-charcoal-700">
          {t('upload.dropzone_text')}
        </p>
        <p className="text-[11px] text-charcoal-400 mt-1">
          Hedef: <span className="font-mono text-charcoal-600 font-medium">{currentDir}</span>
        </p>
      </div>

      {/* Upload queue list */}
      {uploads.length > 0 && (
        <div className="overflow-y-auto space-y-2 mb-4 pr-1 max-h-60 min-h-[100px]">
          {uploads.map((item, idx) => (
            <div
              key={idx}
              className="p-2.5 bg-warm-100 border border-charcoal-200/80 rounded-lg text-xs flex flex-col space-y-1.5"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-charcoal-800 truncate max-w-[260px]" title={item.file.name}>
                  {item.file.name}
                </span>
                <div className="flex items-center space-x-2">
                  <span className="text-charcoal-500">{formatBytes(item.file.size)}</span>
                  {item.status === 'completed' && <CheckIcon className="w-4 h-4 text-accent" />}
                  {item.status === 'uploading' && <SpinnerIcon className="w-3.5 h-3.5 text-accent animate-spin" />}
                  {(item.status === 'pending' || item.status === 'conflict') && !isUploading && (
                    <button
                      type="button"
                      onClick={() => removePending(idx)}
                      className="text-charcoal-400 hover:text-red-500"
                      title={t('upload.remove_from_list')}
                    >
                      <CloseIcon className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
              </div>

              {item.status === 'uploading' && (
                <div className="w-full bg-charcoal-200 rounded-full h-1.5 overflow-hidden">
                  <div
                    className="bg-accent h-1.5 rounded-full transition-all duration-150"
                    style={{ width: `${item.progress}%` }}
                  />
                </div>
              )}

              {item.status === 'conflict' && (
                <div className="flex items-center justify-between bg-yellow-50 p-2 rounded border border-yellow-200 text-yellow-800">
                  <span>{item.error}</span>
                  <button
                    type="button"
                    onClick={() => handleExplicitOverwrite(idx)}
                    disabled={isUploading}
                    className="px-2 py-0.5 bg-yellow-600 hover:bg-yellow-700 text-white rounded font-medium text-[11px]"
                  >
                    {t('upload.overwrite')}
                  </button>
                </div>
              )}

              {item.status === 'error' && (
                <span className="text-red-600 font-medium">{item.error || t('common.error')}</span>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Footer actions */}
      <div className="flex items-center justify-between pt-3 border-t border-charcoal-100">
        <span className="text-xs text-charcoal-500">
          {uploads.length > 0
            ? t('upload.files_selected', { count: uploads.length })
            : t('upload.no_files_selected')}
        </span>
        <div className="flex space-x-2">
          <button
            type="button"
            onClick={onClose}
            disabled={isUploading}
            className="px-3.5 py-2 text-sm font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg transition-colors"
          >
            {allFinished ? 'Kapat' : t('common.cancel')}
          </button>
          {uploads.some((u) => u.status === 'pending') && (
            <button
              type="button"
              onClick={startUploads}
              disabled={isUploading}
              className="px-4 py-2 text-sm font-semibold bg-accent hover:bg-accent-hover text-white rounded-lg shadow-xs transition-colors disabled:opacity-50"
            >
              {isUploading ? t('upload.uploading') : 'Yüklemeyi Başlat'}
            </button>
          )}
        </div>
      </div>
    </Modal>
  );
};
