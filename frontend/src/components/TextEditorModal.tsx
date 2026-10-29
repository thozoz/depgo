import React, { useState, useEffect, useRef, useCallback } from 'react';
import { textApi, ApiError } from '../api/client';
import { formatBytes, getUtf8ByteLength, MAX_TEXT_SIZE_BYTES } from '../utils/format';
import { CloseIcon, FileTextIcon, SpinnerIcon, AlertTriangleIcon } from './Icons';
import { describeError } from '../utils/errors';

interface TextEditorModalProps {
  isOpen: boolean;
  filePath: string;
  fileName: string;
  onClose: () => void;
  onSaved: () => void;
}

export const TextEditorModal: React.FC<TextEditorModalProps> = ({
  isOpen,
  filePath,
  fileName,
  onClose,
  onSaved,
}) => {
  const [content, setContent] = useState('');
  const [initialContent, setInitialContent] = useState('');
  const [revision, setRevision] = useState<number>(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [conflict, setConflict] = useState<string | null>(null);

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const titleId = React.useId();

  const isDirty = content !== initialContent;

  const loadText = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      setConflict(null);
      const data = await textApi.get(filePath);
      setContent(data.content);
      setInitialContent(data.content);
      setRevision(data.revision);
    } catch (err: unknown) {
      setError(describeError(err, 'Metin içeriği yüklenemedi'));
    } finally {
      setLoading(false);
    }
  }, [filePath]);

  useEffect(() => {
    if (isOpen) {
      loadText();
    }
  }, [isOpen, loadText]);

  // Handle beforeunload ONLY when dirty
  useEffect(() => {
    if (!isOpen || !isDirty) return;

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = '';
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => window.removeEventListener('beforeunload', handleBeforeUnload);
  }, [isOpen, isDirty]);

  // Handle Escape key
  const handleAttemptClose = () => {
    if (isDirty) {
      if (!window.confirm('Kaydedilmemiş değişiklikleriniz var. Kapatmak istediğinize emin misiniz?')) {
        return;
      }
    }
    onClose();
  };

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        e.preventDefault();
        handleAttemptClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, isDirty]);

  if (!isOpen) return null;

  const currentByteLength = getUtf8ByteLength(content);
  const isOverSizeLimit = currentByteLength > MAX_TEXT_SIZE_BYTES;

  const handleSave = async () => {
    if (isOverSizeLimit) {
      setError(`Metin boyutu 2 MB sınırını aşıyor (${formatBytes(currentByteLength)}).`);
      return;
    }

    try {
      setSaving(true);
      setError(null);
      setConflict(null);
      const res = await textApi.put(filePath, content, revision);
      setRevision(res.revision);
      setInitialContent(content);
      onSaved();
    } catch (err: unknown) {
      if (err instanceof ApiError && (err.status === 409 || err.status === 412 || err.code === 409 || err.code === 412)) {
        setConflict(
          'Çakışma tespit edildi: Dosya sunucuda başka bir işlem tarafından değiştirilmiş (revizyon uyuşmazlığı). Sunucudaki güncel içeriği yükleyebilirsiniz (kaydedilmemiş değişiklikleriniz kaybolur).'
        );
      } else {
        setError(describeError(err, 'Kayıt işlemi başarısız'));
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4"
    >
      <div
        className="fixed inset-0 bg-charcoal-900/60 backdrop-blur-sm transition-opacity"
        onClick={handleAttemptClose}
        aria-hidden="true"
      />
      <div className="relative bg-white rounded-2xl shadow-2xl border border-charcoal-200 max-w-4xl w-full h-[90vh] z-10 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-charcoal-200 bg-warm-100 flex-shrink-0">
          <div className="flex items-center space-x-2 truncate">
            <FileTextIcon className="w-5 h-5 text-accent flex-shrink-0" />
            <span id={titleId} className="font-semibold text-charcoal-900 text-sm truncate" title={fileName}>
              {fileName}
            </span>
            <span className="text-xs text-charcoal-500 font-mono">
              (Rev: {revision})
            </span>
            {isDirty && (
              <span className="text-[11px] px-2 py-0.5 rounded-full bg-yellow-100 text-yellow-800 font-medium">
                Değiştirildi
              </span>
            )}
          </div>
          <div className="flex items-center space-x-2">
            <span
              className={`text-xs ${
                isOverSizeLimit ? 'text-red-600 font-bold' : 'text-charcoal-500'
              }`}
            >
              {formatBytes(currentByteLength)} / 2 MB
            </span>
            <button
              type="button"
              onClick={handleSave}
              disabled={loading || saving || !isDirty || isOverSizeLimit}
              className="flex items-center space-x-1.5 px-3 py-1.5 bg-accent hover:bg-accent-hover text-white text-xs font-medium rounded-lg shadow-sm transition-colors disabled:opacity-40"
            >
              {saving ? <SpinnerIcon className="w-3.5 h-3.5 animate-spin" /> : null}
              <span>{saving ? 'Kaydediliyor...' : 'Kaydet'}</span>
            </button>
            <button
              type="button"
              onClick={handleAttemptClose}
              disabled={saving}
              className="p-1 text-charcoal-400 hover:text-charcoal-700 rounded-lg transition-colors"
              aria-label="Kapat"
            >
              <CloseIcon className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Conflict banner */}
        {conflict && (
          <div className="p-3 bg-yellow-50 border-b border-yellow-200 text-yellow-900 text-xs flex items-center justify-between gap-2 flex-shrink-0">
            <div className="flex items-center space-x-2">
              <AlertTriangleIcon className="w-4 h-4 text-yellow-600 flex-shrink-0" />
              <span>{conflict}</span>
            </div>
            <button
              type="button"
              onClick={loadText}
              className="px-2.5 py-1 bg-yellow-600 hover:bg-yellow-700 text-white rounded font-medium text-xs whitespace-nowrap"
            >
              Sunucudan Yeniden Yükle
            </button>
          </div>
        )}

        {/* Error banner */}
        {error && (
          <div className="p-3 bg-red-50 border-b border-red-200 text-red-700 text-xs flex items-center justify-between flex-shrink-0">
            <span>{error}</span>
          </div>
        )}

        {/* Text editor body: plain text ONLY */}
        <div className="flex-1 p-3 bg-white flex flex-col overflow-hidden">
          {loading ? (
            <div className="flex-1 flex flex-col items-center justify-center text-charcoal-500 space-y-2">
              <SpinnerIcon className="w-7 h-7 text-accent animate-spin" />
              <span className="text-sm">Metin yükleniyor...</span>
            </div>
          ) : (
            <textarea
              ref={textareaRef}
              value={content}
              onChange={(e) => setContent(e.target.value)}
              disabled={saving}
              aria-label="Metin düzenleyici içeriği"
              className="w-full h-full p-3 font-mono text-xs sm:text-sm text-charcoal-900 bg-warm-50/50 border border-charcoal-200 rounded-xl resize-none focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent selection:bg-accent/20"
              placeholder="Metin içeriği burada düzenlenir..."
            />
          )}
        </div>
      </div>
    </div>
  );
};
