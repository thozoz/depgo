import React, { useState } from 'react';
import { authApi } from '../api/client';
import { LockIcon, SpinnerIcon } from './Icons';

interface LoginViewProps {
  onLoginSuccess: () => void;
}

export const LoginView: React.FC<LoginViewProps> = ({ onLoginSuccess }) => {
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
      setError(err instanceof Error ? err.message : 'Giriş yapılamadı. Bilgilerinizi kontrol edin.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-warm-50 flex flex-col justify-center items-center p-4">
      <div className="max-w-sm w-full bg-white border border-charcoal-200/90 rounded-2xl shadow-xl p-8 space-y-6">
        {/* Brand header */}
        <div className="text-center space-y-2">
          <div className="w-12 h-12 rounded-2xl bg-accent/15 text-accent flex items-center justify-center font-bold text-2xl mx-auto shadow-inner">
            D
          </div>
          <h1 className="text-xl font-bold text-charcoal-900 tracking-tight">Depgo</h1>
          <p className="text-xs text-charcoal-500">Güvenli Yerel Dosya Yönetimi ve Paylaşım</p>
        </div>

        {error && (
          <div
            role="alert"
            className="p-3 bg-red-50 border border-red-200 rounded-xl text-xs text-red-700 space-y-1"
          >
            <div className="font-semibold">Giriş Başarısız</div>
            <div>{error}</div>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="username-input" className="block text-xs font-semibold text-charcoal-700 mb-1.5">
              Kullanıcı Adı
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
              Şifre
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
                <span>Giriş Yapılıyor...</span>
              </>
            ) : (
              <>
                <LockIcon className="w-4 h-4" />
                <span>Giriş Yap</span>
              </>
            )}
          </button>
        </form>

        <div className="pt-2 text-center text-xs text-charcoal-400 border-t border-charcoal-100">
          <span>Yalnızca yönetici girişi desteklenir.</span>
        </div>
      </div>
    </div>
  );
};
