import React, { useState, useEffect } from 'react';
import { FileItem } from '../types';
import { Modal } from './Modal';
import { MoveIcon } from './Icons';
import { joinPath } from '../utils/format';

interface MoveModalProps {
  isOpen: boolean;
  item: FileItem | null;
  currentDir: string;
  onClose: () => void;
  onSubmit: (item: FileItem, targetDir: string) => Promise<void>;
}

export const MoveModal: React.FC<MoveModalProps> = ({
  isOpen,
  item,
  currentDir,
  onClose,
  onSubmit,
}) => {
  const [targetDir, setTargetDir] = useState('/');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setTargetDir(currentDir);
      setError(null);
      setLoading(false);
    }
  }, [isOpen, currentDir]);

  if (!isOpen || !item) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const normalizedTarget = joinPath(targetDir.trim());

    if (normalizedTarget === joinPath(currentDir)) {
      setError('Hedef klasör mevcut klasör ile aynı olamaz');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      await onSubmit(item, normalizedTarget);
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Taşıma işlemi başarısız');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Öğeyi Taşı"
      icon={<MoveIcon className="w-5 h-5 text-accent" />}
    >
      <p className="text-sm text-charcoal-600 mb-4 truncate">
        Taşınacak: <span className="font-semibold text-charcoal-900">{item.name}</span>
      </p>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="target-dir-input" className="block text-xs font-medium text-charcoal-700 mb-1">
            Hedef Klasör Yolu
          </label>
          <input
            id="target-dir-input"
            type="text"
            value={targetDir}
            onChange={(e) => setTargetDir(e.target.value)}
            placeholder="/ veya /klasor-adi"
            className="w-full px-3 py-2 border border-charcoal-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent font-mono text-xs"
            disabled={loading}
          />
          <div className="flex items-center space-x-2 mt-2">
            <button
              type="button"
              onClick={() => setTargetDir('/')}
              className="text-xs px-2 py-1 bg-charcoal-100 hover:bg-charcoal-200 rounded text-charcoal-700 transition-colors"
            >
              Kök Dizin (/)
            </button>
            {currentDir !== '/' && (
              <button
                type="button"
                onClick={() => {
                  const parts = currentDir.split('/').filter(Boolean);
                  parts.pop();
                  setTargetDir('/' + parts.join('/'));
                }}
                className="text-xs px-2 py-1 bg-charcoal-100 hover:bg-charcoal-200 rounded text-charcoal-700 transition-colors"
              >
                Üst Klasör (..)
              </button>
            )}
          </div>
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
            disabled={loading || !targetDir.trim()}
            className="px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg shadow-sm transition-colors disabled:opacity-50"
          >
            {loading ? 'Taşınıyor...' : 'Taşı'}
          </button>
        </div>
      </form>
    </Modal>
  );
};
