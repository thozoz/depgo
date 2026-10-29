/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files (FilesNavigationList.vue)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

import React, { useEffect, useState } from 'react';
import { FolderIcon, ShareIcon, TrashIcon, CloseIcon } from './Icons';
import { useTranslation } from '../utils/i18n';
import { usageApi } from '../api/client';
import { UsageResp } from '../types';
import { formatBytes } from '../utils/format';

interface SidebarProps {
  activeTab: 'files' | 'shares' | 'trash';
  onSelectTab: (tab: 'files' | 'shares' | 'trash') => void;
  isOpenMobile: boolean;
  onCloseMobile: () => void;
  // Changes whenever the file list reloads, so the usage figures follow uploads and deletes.
  usageRefreshToken?: unknown;
}

export const Sidebar: React.FC<SidebarProps> = ({
  activeTab,
  onSelectTab,
  isOpenMobile,
  onCloseMobile,
  usageRefreshToken,
}) => {
  const { t } = useTranslation();
  const [usage, setUsage] = useState<UsageResp | null>(null);

  useEffect(() => {
    let cancelled = false;
    usageApi
      .get()
      .then((u) => {
        if (!cancelled) setUsage(u);
      })
      .catch(() => {
        if (!cancelled) setUsage(null);
      });
    return () => {
      cancelled = true;
    };
  }, [usageRefreshToken]);

  const usedPercent = usage && usage.total > 0 ? Math.min(100, (usage.used / usage.total) * 100) : 0;
  const nearlyFull = usage ? usage.total > 0 && usage.free / usage.total < 0.1 : false;

  const navItems = [
    {
      id: 'files' as const,
      name: t('nav.files'),
      icon: FolderIcon,
    },
    {
      id: 'shares' as const,
      name: t('nav.shares'),
      icon: ShareIcon,
    },
    {
      id: 'trash' as const,
      name: t('nav.trash'),
      icon: TrashIcon,
    },
  ];

  const content = (
    <div className="flex flex-col h-full bg-white border-r border-charcoal-200 w-60 select-none">
      <div className="flex items-center justify-between p-4 md:hidden border-b border-charcoal-100">
        <span className="font-semibold text-charcoal-900 text-sm">{t('common.menu')}</span>
        <button
          type="button"
          onClick={onCloseMobile}
          className="p-1 text-charcoal-500 hover:text-charcoal-800 rounded-md"
          aria-label={t('common.close')}
        >
          <CloseIcon className="w-5 h-5" />
        </button>
      </div>

      {/* Navigation links - Nextcloud FilesNavigationList layout */}
      <nav aria-label={t('nav.main_menu')} className="flex-1 p-3 space-y-1 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = activeTab === item.id;
          return (
            <button
              key={item.id}
              type="button"
              onClick={() => {
                onSelectTab(item.id);
                onCloseMobile();
              }}
              className={`w-full flex items-center space-x-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all ${
                isActive
                  ? 'bg-accent/10 text-accent font-semibold border-l-4 border-accent rounded-l-none'
                  : 'text-charcoal-700 hover:bg-charcoal-50 hover:text-charcoal-900'
              }`}
            >
              <Icon className={`w-4 h-4 flex-shrink-0 ${isActive ? 'text-accent' : 'text-charcoal-500'}`} />
              <span className="truncate">{item.name}</span>
            </button>
          );
        })}
      </nav>

      {/* Footer quota - Nextcloud NavigationQuota style */}
      <div className="p-4 border-t border-charcoal-200 text-xs text-charcoal-500">
        <div className="flex items-center justify-between mb-1.5">
          <span className="font-medium text-charcoal-700">{t('nav.storage_quota')}</span>
          <span className="text-[11px] text-accent font-medium">Depgo Local</span>
        </div>
        {usage ? (
          <>
            <div
              className="w-full bg-charcoal-100 h-1.5 rounded-full overflow-hidden"
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={Math.round(usedPercent)}
              aria-label={t('nav.storage_quota')}
            >
              <div
                className={`h-full rounded-full ${nearlyFull ? 'bg-red-500' : 'bg-accent'}`}
                style={{ width: `${Math.max(usedPercent, usage.used > 0 ? 1 : 0)}%` }}
              />
            </div>
            <p className="mt-1.5 text-charcoal-700">
              {t('nav.storage_used_of', { used: formatBytes(usage.used), total: formatBytes(usage.total) })}
            </p>
            <p className="text-[11px] text-charcoal-500">
              {t('nav.storage_free', { free: formatBytes(usage.free) })}
            </p>
          </>
        ) : (
          <div className="w-full bg-charcoal-100 h-1.5 rounded-full" />
        )}
      </div>
    </div>
  );

  return (
    <>
      {/* Desktop sidebar */}
      <aside className="hidden md:flex flex-shrink-0">
        {content}
      </aside>

      {/* Mobile drawer */}
      {isOpenMobile && (
        <div className="fixed inset-0 z-30 flex md:hidden">
          <div
            className="fixed inset-0 bg-charcoal-900/40 backdrop-blur-sm"
            onClick={onCloseMobile}
            aria-hidden="true"
          />
          <div className="relative z-40 flex-1 max-w-xs w-full">
            {content}
          </div>
        </div>
      )}
    </>
  );
};
