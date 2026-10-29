import React, { useState, useEffect } from 'react';
import { FileItem } from '../types';
import { Modal } from './Modal';
import { MoveIcon } from './Icons';
import { joinPath } from '../utils/format';
import { describeError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

interface MoveModalProps {
  isOpen: boolean;
  item: FileItem | null;
  items?: FileItem[];
  currentDir: string;
  onClose: () => void;
  onSubmit: (targetItems: FileItem[], targetDir: string) => Promise<void>;
}

export const MoveModal: React.FC<MoveModalProps> = ({
  isOpen,
  item,
  items,
  currentDir,
  onClose,
  onSubmit,
}) => {
  const { t } = useTranslation();
  const [targetDir, setTargetDir] = useState('/');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const targets = items && items.length > 0 ? items : item ? [item] : [];

  useEffect(() => {
    if (isOpen) {
      setTargetDir(currentDir);
      setError(null);
      setLoading(false);
    }
  }, [isOpen, currentDir]);

  if (!isOpen || targets.length === 0) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const normalizedTarget = joinPath(targetDir.trim());

    if (normalizedTarget === joinPath(currentDir)) {
      setError(t('move.same_folder_error'));
      return;
    }

    try {
      setLoading(true);
      setError(null);
      await onSubmit(targets, normalizedTarget);
      onClose();
    } catch (err: unknown) {
      setError(describeError(err, t('files.move_error')));
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
      title={t('move.title')}
      icon={<MoveIcon className="w-5 h-5 text-accent" />}
    >
      <div className="text-sm text-charcoal-600 mb-4 truncate">
        {t('move.moving_item')}{' '}
        <span className="font-semibold text-charcoal-900">
          {isMultiple ? t('files.item_count', { count: targets.length }) : single.name}
        </span>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="target-dir" className="block text-xs font-medium text-charcoal-700 mb-1">
            {t('move.target_dir_label')}
          </label>
          <input
            id="target-dir"
            type="text"
            value={targetDir}
            onChange={(e) => setTargetDir(e.target.value)}
            disabled={loading}
            placeholder="/"
            className="w-full px-3 py-2 text-sm border border-charcoal-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-accent/40 focus:border-accent"
          />
          <p className="mt-1 text-xs text-charcoal-400">{t('move.example_path')}</p>
        </div>

        {error && (
          <div className="p-2.5 rounded-lg bg-red-50 text-red-700 text-xs border border-red-200">
            {error}
          </div>
        )}

        <div className="flex justify-end space-x-2 pt-2">
          <button
            type="button"
            onClick={onClose}
            disabled={loading}
            className="px-3.5 py-2 text-sm font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg transition-colors"
          >
            {t('common.cancel')}
          </button>
          <button
            type="submit"
            disabled={loading}
            className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg shadow-sm transition-colors disabled:opacity-50"
          >
            {loading ? t('common.loading') : t('common.move')}
          </button>
        </div>
      </form>
    </Modal>
  );
};
