import React from 'react';
import { User } from '../types';
import { GridIcon, ListIcon, LogoutIcon, MenuIcon } from './Icons';

interface NavbarProps {
  user: User | null;
  viewMode: 'list' | 'grid';
  onToggleViewMode: () => void;
  onLogout: () => void;
  onOpenMobileMenu?: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({
  user,
  viewMode,
  onToggleViewMode,
  onLogout,
  onOpenMobileMenu,
}) => {
  return (
    <header className="h-14 bg-white border-b border-charcoal-200 px-4 flex items-center justify-between sticky top-0 z-20">
      <div className="flex items-center space-x-3">
        {onOpenMobileMenu && (
          <button
            type="button"
            onClick={onOpenMobileMenu}
            className="p-1.5 text-charcoal-600 hover:text-charcoal-900 rounded-lg hover:bg-charcoal-100 md:hidden"
            aria-label="Menüyü aç"
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
        <button
          type="button"
          onClick={onToggleViewMode}
          className="p-2 text-charcoal-600 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-lg transition-colors"
          title={viewMode === 'list' ? 'Izgara Görünümü' : 'Liste Görünümü'}
          aria-label={viewMode === 'list' ? 'Izgara görünümüne geç' : 'Liste görünümüne geç'}
        >
          {viewMode === 'list' ? <GridIcon className="w-4 h-4" /> : <ListIcon className="w-4 h-4" />}
        </button>

        <div className="h-5 w-px bg-charcoal-200 mx-1" />

        {user && (
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-charcoal-700 hidden sm:inline-block">
              {user.username}
            </span>
            <button
              type="button"
              onClick={onLogout}
              className="flex items-center space-x-1.5 px-2.5 py-1.5 text-sm text-charcoal-600 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
              title="Çıkış Yap"
            >
              <LogoutIcon className="w-4 h-4" />
              <span className="hidden sm:inline">Çıkış</span>
            </button>
          </div>
        )}
      </div>
    </header>
  );
};
