import React from 'react';
import { User } from '../types';
import { LogoutIcon, MenuIcon } from './Icons';
import { useTranslation } from '../utils/i18n';

interface NavbarProps {
  user: User | null;
  onOpenAccount: () => void;
  onLogout: () => void;
  onOpenMobileMenu?: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  user,
  onOpenAccount,
  onLogout,
  onOpenMobileMenu,
}) => {
  const { t, lang, setLanguage } = useTranslation();

  return (
    <header className="h-14 bg-white border-b border-charcoal-200 px-4 flex items-center justify-between sticky top-0 z-20">
      <div className="flex items-center space-x-3">
        {onOpenMobileMenu && (
          <button
            type="button"
            onClick={onOpenMobileMenu}
            className="p-1.5 text-charcoal-600 hover:text-charcoal-900 rounded-lg hover:bg-charcoal-100 md:hidden"
            aria-label={t('nav.open_menu')}
          >
            <MenuIcon className="w-5 h-5" />
          </button>
        )}
        <div className="flex items-center space-x-2">
          <div className="w-8 h-8 rounded-lg bg-accent/15 text-accent flex items-center justify-center font-bold text-lg">
            D
          </div>
          <span className="font-semibold text-charcoal-900 text-lg tracking-tight">Depgo</span>
        </div>
      </div>

      <div className="flex items-center space-x-2">

        {/* Language switch */}
        <div className="flex items-center rounded-lg border border-charcoal-200 p-0.5 text-xs font-medium bg-warm-50 text-charcoal-600">
          <button
            type="button"
            onClick={() => setLanguage('tr')}
            className={`px-1.5 py-0.5 rounded transition-all ${
              lang === 'tr' ? 'bg-white font-semibold text-charcoal-900 shadow-sm' : 'hover:text-charcoal-900'
            }`}
            aria-label={t('nav.lang_tr')}
          >
            TR
          </button>
          <button
            type="button"
            onClick={() => setLanguage('en')}
            className={`px-1.5 py-0.5 rounded transition-all ${
              lang === 'en' ? 'bg-white font-semibold text-charcoal-900 shadow-sm' : 'hover:text-charcoal-900'
            }`}
            aria-label={t('nav.lang_en')}
          >
            EN
          </button>
        </div>

        {user && (
          <div className="flex items-center space-x-2">
            <button
              type="button"
              onClick={onOpenAccount}
              className="text-sm font-medium text-charcoal-700 hover:text-accent hover:bg-charcoal-100 px-2 py-1.5 rounded-lg transition-colors"
              title={t('account.title')}
              aria-label={t('account.title')}
            >
              {user.username}
            </button>
            <button
              type="button"
              onClick={onLogout}
              className="flex items-center space-x-1.5 px-2.5 py-1.5 text-sm text-charcoal-600 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
              title={t('nav.logout')}
            >
              <LogoutIcon className="w-4 h-4" />
              <span className="hidden sm:inline">{t('nav.logout')}</span>
            </button>
          </div>
        )}
      </div>
    </header>
  );
};
