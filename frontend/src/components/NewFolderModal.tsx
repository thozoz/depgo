import React, { useState, useEffect } from 'react';
import { Modal } from './Modal';
import { FolderIcon } from './Icons';

interface NewFolderModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (folderName: string) => Promise<void>;
}

export const NewFolderModal: React.FC<NewFolderModalProps> = ({ isOpen, onClose, onSubmit }) => {
  const [name, setName] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setName('');
      setError(null);
      setLoading(false);
    }
  }, [isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) {
      setError('Klasör adı boş bırakılamaz');
      return;
    }
    if (trimmed.includes('/') || trimmed.includes('\\')) {
      setError('Klasör adı / veya \\ içeremez');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      await onSubmit(trimmed);
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Klasör oluşturulamadı');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Yeni Klasör Oluştur"
      icon={<FolderIcon className="w-5 h-5 text-accent" />}
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="folder-name-input" className="block text-xs font-medium text-charcoal-700 mb-1">
            Klasör Adı
          </label>
          <input
            id="folder-name-input"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Örn. Projeler"
            className="w-full px-3 py-2 border border-charcoal-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent"
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
            İptal
          </button>
          <button
            type="submit"
            disabled={loading || !name.trim()}
            className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg shadow-sm transition-colors disabled:opacity-50"
          >
            {loading ? 'Oluşturuluyor...' : 'Oluştur'}
          </button>
        </div>
      </form>
    </Modal>
  );
};
