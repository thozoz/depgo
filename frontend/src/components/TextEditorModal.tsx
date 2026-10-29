import React, { useState, useEffect, useRef, useCallback } from 'react';
import { textApi, ApiError } from '../api/client';
import { formatBytes, getUtf8ByteLength, MAX_TEXT_SIZE_BYTES } from '../utils/format';
import { CloseIcon, FileTextIcon, SpinnerIcon, AlertTriangleIcon } from './Icons';
import { describeError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

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
  const { t } = useTranslation();
  const [content, setContent] = useState('');
  const [initialContent, setInitialContent] = useState('');
  const [revision, setRevision] = useState<number>(0);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [conflict, setConflict] = useState<string | null>(null);

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const backdropPressRef = useRef(false);
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
      setError(describeError(err, t('editor.load_failed')));
    } finally {
      setLoading(false);
    }
  }, [filePath, t]);

  useEffect(() => {
    if (isOpen) {
      loadText();
    }
  }, [isOpen, loadText]);

  useEffect(() => {
    if (!isOpen || !isDirty) return;

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = '';
    };

    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => window.removeEventListener('beforeunload', handleBeforeUnload);
  }, [isOpen, isDirty]);

  const handleAttemptClose = () => {
    if (isDirty) {
      if (!window.confirm(t('editor.unsaved_warning'))) {
        return;
      }
    }
    onClose();
  };

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        handleAttemptClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, isDirty]);

  const handleSave = async () => {
    const byteLength = getUtf8ByteLength(content);
    if (byteLength > MAX_TEXT_SIZE_BYTES) {
      setError(t('editor.size_limit_exceeded_editor', { formattedSize: formatBytes(byteLength) }));
      return;
    }

    try {
      setSaving(true);
      setError(null);
      setConflict(null);
      const resp = await textApi.put(filePath, content, revision);
      setRevision(resp.revision);
      setInitialContent(content);
      onSaved();
    } catch (err: unknown) {
      if (err instanceof ApiError && (err.status === 409 || err.code === 409)) {
        setConflict(t('editor.conflict_desc'));
      } else {
        setError(describeError(err, t('editor.save_failed')));
      }
    } finally {
      setSaving(false);
    }
  };

  const handleReload = async () => {
    if (isDirty && !window.confirm(t('editor.unsaved_reload_warning'))) {
      return;
    }
    await loadText();
  };

  if (!isOpen) return null;

  const byteLength = getUtf8ByteLength(content);

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="fixed inset-0 z-50 flex items-center justify-center p-2 sm:p-4 bg-charcoal-900/60 backdrop-blur-sm"
      onMouseDown={(e) => {
        backdropPressRef.current = e.target === e.currentTarget;
      }}
      onClick={(e) => {
        // Only a press that started on the backdrop closes it, so dragging a text
        // selection out of the textarea and releasing outside does not.
        if (e.target === e.currentTarget && backdropPressRef.current) handleAttemptClose();
      }}
    >
      <div className="bg-white rounded-2xl shadow-2xl border border-charcoal-200 w-full max-w-4xl h-[90vh] flex flex-col overflow-hidden">
        {/* Header */}
        <div className="px-5 py-3.5 border-b border-charcoal-200 flex items-center justify-between bg-warm-50/70">
          <div className="flex items-center space-x-2.5 min-w-0">
            <FileTextIcon className="w-5 h-5 text-emerald-600 flex-shrink-0" />
            <h2 id={titleId} className="text-sm font-semibold text-charcoal-900 truncate">
              {fileName}
            </h2>
            <span className="font-mono text-xs px-2 py-0.5 rounded bg-charcoal-100 text-charcoal-600">
              Rev #{revision}
            </span>
            {isDirty && (
              <span className="text-[11px] font-semibold px-2 py-0.5 rounded bg-yellow-100 text-yellow-800">
                {t('editor.modified_badge')}
              </span>
            )}
          </div>
          <button
            type="button"
            onClick={handleAttemptClose}
            className="p-1.5 text-charcoal-500 hover:text-charcoal-800 rounded-lg hover:bg-charcoal-100 transition-colors"
            aria-label={t('common.close')}
          >
            <CloseIcon className="w-5 h-5" />
          </button>
        </div>

        {/* Conflict / Error alerts */}
        {conflict && (
          <div className="p-3 bg-yellow-50 border-b border-yellow-200 text-xs text-yellow-900 flex items-start justify-between">
            <div className="flex items-start space-x-2">
              <AlertTriangleIcon className="w-4 h-4 text-yellow-700 flex-shrink-0 mt-0.5" />
              <span>{conflict}</span>
            </div>
            <button
              type="button"
              onClick={handleReload}
              className="ml-3 px-2 py-1 bg-yellow-600 hover:bg-yellow-700 text-white rounded font-medium text-xs flex-shrink-0"
            >
              {t('editor.reload')}
            </button>
          </div>
        )}

        {error && (
          <div className="p-3 bg-red-50 border-b border-red-200 text-xs text-red-700 flex items-center justify-between">
            <span>{error}</span>
            <button
              type="button"
              onClick={() => setError(null)}
              className="text-red-500 hover:text-red-700 text-xs font-medium"
            >
              {t('common.close')}
            </button>
          </div>
        )}

        {/* Content Area */}
        <div className="flex-1 relative bg-charcoal-50/30 overflow-hidden flex flex-col">
          {loading ? (
            <div className="flex-1 flex flex-col items-center justify-center text-charcoal-500">
              <SpinnerIcon className="w-8 h-8 text-accent animate-spin mb-2" />
              <span className="text-sm">{t('editor.loading')}</span>
            </div>
          ) : (
            <textarea
              ref={textareaRef}
              value={content}
              onChange={(e) => setContent(e.target.value)}
              aria-label={t('editor.content_label')}
              className="w-full flex-1 p-4 font-mono text-xs sm:text-sm text-charcoal-900 bg-white resize-none border-0 focus:outline-none focus:ring-0 leading-relaxed overflow-y-auto"
              placeholder={t('editor.empty_placeholder')}
              spellCheck={false}
            />
          )}
        </div>

        {/* Footer */}
        <div className="px-5 py-3 border-t border-charcoal-200 flex items-center justify-between bg-warm-50/70 text-xs text-charcoal-500">
          <div className="flex items-center space-x-4">
            <span>
              {t('files.size')}: <span className="font-mono text-charcoal-700">{formatBytes(byteLength)}</span>
            </span>
            <span>
              {t('editor.line_count')}:{' '}
              <span className="font-mono text-charcoal-700">{content.split('\n').length}</span>
            </span>
          </div>

          <div className="flex items-center space-x-2">
            <button
              type="button"
              onClick={handleAttemptClose}
              disabled={saving}
              className="px-3 py-1.5 text-xs font-medium text-charcoal-700 hover:bg-charcoal-100 rounded-lg transition-colors"
            >
              {t('common.cancel')}
            </button>
            <button
              type="button"
              onClick={handleSave}
              disabled={saving || loading || !isDirty}
              className="px-4 py-1.5 text-xs font-semibold bg-accent hover:bg-accent-hover text-white rounded-lg shadow-xs transition-colors flex items-center space-x-1.5 disabled:opacity-40"
            >
              {saving ? (
                <>
                  <SpinnerIcon className="w-3.5 h-3.5 animate-spin" />
                  <span>{t('common.saving')}</span>
                </>
              ) : (
                <span>{t('common.save')}</span>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
