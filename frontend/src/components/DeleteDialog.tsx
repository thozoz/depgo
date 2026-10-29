import React, { useState } from 'react';
import { FileItem } from '../types';
import { Modal } from './Modal';
import { TrashIcon } from './Icons';
import { describeError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

interface DeleteDialogProps {
  isOpen: boolean;
  item: FileItem | null;
  items?: FileItem[];
  onClose: () => void;
  onConfirm: (targetItems: FileItem[]) => Promise<void>;
}

export const DeleteDialog: React.FC<DeleteDialogProps> = ({
  isOpen,
  item,
  items,
  onClose,
  onConfirm,
}) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const targets = items && items.length > 0 ? items : item ? [item] : [];

  if (!isOpen || targets.length === 0) return null;

  const handleDelete = async () => {
    try {
      setLoading(true);
      setError(null);
      await onConfirm(targets);
      onClose();
    } catch (err: unknown) {
      setError(describeError(err, t('delete.failed')));
    } finally {
      setLoading(false);
    }
  };

  const isMultiple = targets.length > 1;
  const single = targets[0];

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t('delete.title')}
      icon={<TrashIcon className="w-5 h-5 text-red-600" />}
      maxWidthClass="max-w-sm"
    >
      <div className="text-sm text-charcoal-600 mb-4 break-words">
        {isMultiple ? (
          <p>{t('delete.multiple_confirm', { count: targets.length })}</p>
        ) : (
          <p>
            {t('delete.single_confirm', {
              name: single.name,
              type: single.is_dir ? t('delete.folder') : t('delete.file'),
            })}
          </p>
        )}
      </div>

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
          {t('common.cancel')}
        </button>
        <button
          type="button"
          onClick={handleDelete}
          disabled={loading}
          className="px-4 py-2 text-sm font-medium bg-red-600 hover:bg-red-700 text-white rounded-lg shadow-sm transition-colors disabled:opacity-50"
        >
          {loading ? t('common.loading') : t('common.delete')}
        </button>
      </div>
    </Modal>
  );
};
