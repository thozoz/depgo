import React, { useState } from 'react';
import { FileItem } from '../types';
import { Modal } from './Modal';
import { TrashIcon } from './Icons';

interface DeleteDialogProps {
  isOpen: boolean;
  item: FileItem | null;
  onClose: () => void;
  onConfirm: (item: FileItem) => Promise<void>;
}

export const DeleteDialog: React.FC<DeleteDialogProps> = ({
  isOpen,
  item,
  onClose,
  onConfirm,
}) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen || !item) return null;

  const handleDelete = async () => {
    try {
      setLoading(true);
      setError(null);
      await onConfirm(item);
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Silme işlemi başarısız');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Öğeyi Sil"
      icon={<TrashIcon className="w-5 h-5 text-red-600" />}
      maxWidthClass="max-w-sm"
    >
      <p className="text-sm text-charcoal-600 mb-4 break-words">
        <span className="font-semibold text-charcoal-900">{item.name}</span>{' '}
        {item.is_dir ? 'klasörünü' : 'dosyasını'} silmek istediğinize emin misiniz?
      </p>

      {error && (
        <div className="p-2.5 rounded-lg bg-red-50 text-red-700 text-xs border border-red-200 mb-4">
          {error}
        </div>
      )}

      <div className="flex justify-end space-x-2">
        <button
          type="button"
          onClick={onClose}
          disabled={loading}
          className="px-3.5 py-2 text-sm font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg transition-colors"
        >
          Vazgeç
        </button>
        <button
          type="button"
          onClick={handleDelete}
          disabled={loading}
          className="px-4 py-2 text-sm font-medium bg-red-600 hover:bg-red-700 text-white rounded-lg shadow-sm transition-colors disabled:opacity-50"
        >
          {loading ? 'Siliniyor...' : 'Sil'}
        </button>
      </div>
    </Modal>
  );
};
