/**
 * SPDX-FileCopyrightText: 2026 Depgo contributors, inspired by Nextcloud Files (BreadCrumbs.vue)
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

import React from 'react';
import { getBreadcrumbs } from '../utils/format';
import { ChevronRightIcon, FolderIcon } from './Icons';
import { useTranslation } from '../utils/i18n';

interface BreadcrumbsProps {
  currentPath: string;
  onNavigate: (path: string) => void;
  onDropToPath?: (targetPath: string, e: React.DragEvent) => void;
  dragOverPath?: string | null;
}

export const Breadcrumbs: React.FC<BreadcrumbsProps> = ({
  currentPath,
  onNavigate,
  onDropToPath,
  dragOverPath,
}) => {
  const { t } = useTranslation();
  const crumbs = getBreadcrumbs(currentPath);

  return (
    <nav aria-label={t('nav.breadcrumb_aria')} className="flex items-center space-x-1 text-sm overflow-x-auto py-1 min-w-0">
      {crumbs.map((crumb, idx) => {
        const isLast = idx === crumbs.length - 1;
        const isRoot = idx === 0;
        const isDropOver = dragOverPath === crumb.path;
        const displayName = isRoot ? t('nav.files') : crumb.name;

        return (
          <React.Fragment key={crumb.path}>
            {idx > 0 && <ChevronRightIcon className="w-3.5 h-3.5 text-charcoal-400 flex-shrink-0" />}
            <button
              type="button"
              onClick={() => onNavigate(crumb.path)}
              disabled={isLast}
              onDragOver={(e) => {
                if (onDropToPath && !isLast) {
                  e.preventDefault();
                  e.dataTransfer.dropEffect = 'move';
                }
              }}
              onDrop={(e) => {
                if (onDropToPath && !isLast) {
                  e.preventDefault();
                  onDropToPath(crumb.path, e);
                }
              }}
              className={`flex items-center space-x-1.5 px-2.5 py-1 rounded-lg text-sm transition-all flex-shrink-0 ${
                isDropOver ? 'ring-2 ring-accent bg-accent/20' : ''
              } ${
                isLast
                  ? 'font-semibold text-charcoal-900 cursor-default bg-charcoal-100/60'
                  : 'text-charcoal-600 hover:text-charcoal-900 hover:bg-charcoal-100'
              }`}
              title={displayName}
            >
              {isRoot && <FolderIcon className="w-4 h-4 text-accent flex-shrink-0" />}
              <span className="truncate max-w-[140px] sm:max-w-xs">{displayName}</span>
            </button>
          </React.Fragment>
        );
      })}
    </nav>
  );
};
