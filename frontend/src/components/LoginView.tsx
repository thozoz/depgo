import React, { useState } from 'react';
import { authApi } from '../api/client';
import { LockIcon, SpinnerIcon } from './Icons';
import { describeError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

interface LoginViewProps {
  onLoginSuccess: () => void;
}

export const LoginView: React.FC<LoginViewProps> = ({ onLoginSuccess }) => {
  const { t, lang, setLanguage } = useTranslation();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password) {
      setError('Lütfen kullanıcı adı ve şifrenizi girin');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      await authApi.login(username.trim(), password);
      onLoginSuccess();
    } catch (err: unknown) {
      setError(describeError(err, 'Giriş yapılamadı. Bilgilerinizi kontrol edin.'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-warm-50 flex flex-col justify-center items-center p-4 relative">
      {/* Top right language switch */}
      <div className="absolute top-4 right-4 flex items-center rounded-lg border border-charcoal-200 p-0.5 text-xs font-medium bg-white text-charcoal-600 shadow-xs">
        <button
          type="button"
          onClick={() => setLanguage('tr')}
          className={`px-2 py-1 rounded transition-all ${
            lang === 'tr' ? 'bg-warm-100 font-semibold text-charcoal-900 shadow-xs' : 'hover:text-charcoal-900'
          }`}
          aria-label="Türkçe"
        >
          TR
        </button>
        <button
          type="button"
          onClick={() => setLanguage('en')}
          className={`px-2 py-1 rounded transition-all ${
            lang === 'en' ? 'bg-warm-100 font-semibold text-charcoal-900 shadow-xs' : 'hover:text-charcoal-900'
          }`}
          aria-label="English"
        >
          EN
        </button>
      </div>

      <div className="max-w-sm w-full bg-white border border-charcoal-200/90 rounded-2xl shadow-xl p-8 space-y-6">
        {/* Brand header */}
        <div className="text-center space-y-2">
          <div className="w-12 h-12 rounded-2xl bg-accent/15 text-accent flex items-center justify-center font-bold text-2xl mx-auto shadow-inner">
            D
          </div>
          <h1 className="text-xl font-bold text-charcoal-900 tracking-tight">Depgo</h1>
          <p className="text-xs text-charcoal-500">{t('app.tagline')}</p>
        </div>

        {error && (
          <div
            role="alert"
            className="p-3 bg-red-50 border border-red-200 rounded-xl text-xs text-red-700 space-y-1"
          >
            <div className="font-semibold">{t('common.error')}</div>
            <div>{error}</div>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="username-input" className="block text-xs font-semibold text-charcoal-700 mb-1.5">
              {t('login.username_label')}
            </label>
            <input
              id="username-input"
              type="text"
              autoComplete="username"
              required
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="admin"
              disabled={loading}
              className="w-full px-3.5 py-2.5 border border-charcoal-200 rounded-xl text-sm text-charcoal-900 placeholder-charcoal-400 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent transition-colors"
            />
          </div>

          <div>
            <label htmlFor="password-input" className="block text-xs font-semibold text-charcoal-700 mb-1.5">
              {t('login.password_label')}
            </label>
            <input
              id="password-input"
              type="password"
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              disabled={loading}
              className="w-full px-3.5 py-2.5 border border-charcoal-200 rounded-xl text-sm text-charcoal-900 placeholder-charcoal-400 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent transition-colors"
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="w-full py-2.5 bg-accent hover:bg-accent-hover text-white text-sm font-semibold rounded-xl shadow-sm transition-colors flex items-center justify-center space-x-2 disabled:opacity-50"
          >
            {loading ? (
              <>
                <SpinnerIcon className="w-4 h-4 animate-spin" />
                <span>{t('login.submitting')}</span>
              </>
            ) : (
              <>
                <LockIcon className="w-4 h-4" />
                <span>{t('login.submit_button')}</span>
              </>
            )}
          </button>
        </form>
      </div>
    </div>
  );
};
