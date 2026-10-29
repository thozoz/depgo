import React, { useState, useEffect } from 'react';
import { FileItem } from '../types';
import { Modal } from './Modal';
import { EditIcon } from './Icons';
import { describeError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

interface RenameModalProps {
  isOpen: boolean;
  item: FileItem | null;
  onClose: () => void;
  onSubmit: (item: FileItem, newName: string) => Promise<void>;
}

export const RenameModal: React.FC<RenameModalProps> = ({ isOpen, item, onClose, onSubmit }) => {
  const { t } = useTranslation();
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen && item) {
      setName(item.name);
      setError(null);
      setLoading(false);
    }
  }, [isOpen, item]);

  if (!isOpen || !item) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      setError(t('rename.empty_error'));
      return;
    }
    if (trimmed === item.name) {
      onClose();
      return;
    }
    if (trimmed.includes('/') || trimmed.includes('\\')) {
      setError(t('rename.invalid_chars'));
      return;
    }

    try {
      setLoading(true);
      setError(null);
      await onSubmit(item, trimmed);
      onClose();
    } catch (err: unknown) {
      setError(describeError(err, t('rename.failed')));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t('rename.title')}
      icon={<EditIcon className="w-5 h-5 text-accent" />}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="rename-input" className="block text-xs font-semibold text-charcoal-700 mb-1">
            {t('rename.name_label')}
          </label>
          <input
            id="rename-input"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full px-3 py-2 border border-charcoal-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
            disabled={loading}
          />
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
            disabled={loading || !name.trim()}
            className="px-4 py-2 text-sm font-semibold bg-accent hover:bg-accent-hover text-white rounded-lg shadow-xs transition-colors disabled:opacity-50"
          >
            {loading ? t('common.saving') : t('rename.save_button')}
          </button>
        </div>
      </form>
    </Modal>
  );
};
