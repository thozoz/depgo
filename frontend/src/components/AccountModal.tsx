import React, { useEffect, useState } from 'react';
import { User } from '../types';
import { authApi } from '../api/client';
import { Modal } from './Modal';
import { describeError } from '../utils/errors';
import { useTranslation } from '../utils/i18n';

interface AccountModalProps {
  isOpen: boolean;
  user: User | null;
  onClose: () => void;
  onSessionEnded: () => void;
}

const inputClass =
  'w-full px-3 py-2 border border-charcoal-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent';
const buttonClass =
  'px-4 py-2 text-sm font-medium bg-accent hover:bg-accent-hover text-white rounded-lg shadow-sm transition-colors disabled:opacity-50';

export const AccountModal: React.FC<AccountModalProps> = ({ isOpen, user, onClose, onSessionEnded }) => {
  const { t } = useTranslation();
  const [username, setUsername] = useState('');
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (isOpen) {
      setUsername(user?.username ?? '');
      setCurrentPassword('');
      setNewPassword('');
      setConfirmPassword('');
      setError(null);
      setBusy(false);
    }
  }, [isOpen, user]);

  if (!isOpen || !user) return null;

  const run = async (body: { username?: string; password?: string; current_password?: string }) => {
    setBusy(true);
    setError(null);
    try {
      await authApi.updateMe(body);
      // A new name or password invalidates every earlier session, including this one.
      onSessionEnded();
    } catch (err: unknown) {
      setError(describeError(err, t('account.failed')));
    } finally {
      setBusy(false);
    }
  };

  const saveUsername = (e: React.FormEvent) => {
    e.preventDefault();
    const name = username.trim();
    if (!name || name === user.username) return;
    void run({ username: name });
  };

  const savePassword = (e: React.FormEvent) => {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      setError(t('account.password_mismatch'));
      return;
    }
    void run({ password: newPassword, current_password: currentPassword });
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t('account.title')} maxWidthClass="max-w-md">
      <div className="space-y-6">
        <form onSubmit={saveUsername} className="space-y-3">
          <label htmlFor="account-username" className="block text-xs font-medium text-charcoal-700">
            {t('account.username')}
          </label>
          <input id="account-username" type="text" value={username} onChange={(e) => setUsername(e.target.value)} className={inputClass} disabled={busy} autoComplete="username" />
          <p className="text-[11px] text-charcoal-500">{t('account.signout_hint')}</p>
          <div className="flex justify-end">
            <button type="submit" className={buttonClass} disabled={busy || !username.trim() || username.trim() === user.username}>
              {t('account.save_username')}
            </button>
          </div>
        </form>

        <form onSubmit={savePassword} className="space-y-3 pt-4 border-t border-charcoal-100">
          <h3 className="text-xs font-semibold text-charcoal-700 uppercase tracking-wide">{t('account.change_password')}</h3>
          <div>
            <label htmlFor="account-current" className="block text-xs font-medium text-charcoal-700 mb-1">{t('account.current_password')}</label>
            <input id="account-current" type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} className={inputClass} disabled={busy} autoComplete="current-password" />
          </div>
          <div>
            <label htmlFor="account-new" className="block text-xs font-medium text-charcoal-700 mb-1">{t('account.new_password')}</label>
            <input id="account-new" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} className={inputClass} disabled={busy} autoComplete="new-password" />
          </div>
          <div>
            <label htmlFor="account-confirm" className="block text-xs font-medium text-charcoal-700 mb-1">{t('account.confirm_password')}</label>
            <input id="account-confirm" type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} className={inputClass} disabled={busy} autoComplete="new-password" />
          </div>
          <div className="flex justify-end">
            <button type="submit" className={buttonClass} disabled={busy || !currentPassword || !newPassword}>
              {t('account.save_password')}
            </button>
          </div>
        </form>

        {error && <div role="alert" className="p-2.5 rounded-lg bg-red-50 text-red-700 text-xs border border-red-200">{error}</div>}
      </div>
    </Modal>
  );
};
