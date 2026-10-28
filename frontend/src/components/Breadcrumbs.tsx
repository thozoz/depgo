import React from 'react';
import { getBreadcrumbs } from '../utils/format';
import { ChevronRightIcon, FolderIcon } from './Icons';

interface BreadcrumbsProps {
  currentPath: string;
  onNavigate: (path: string) => void;
}

export const Breadcrumbs: React.FC<BreadcrumbsProps> = ({ currentPath, onNavigate }) => {
  const crumbs = getBreadcrumbs(currentPath);

  return (
    <nav aria-label="Klasör Gezintisi" className="flex items-center space-x-1 text-sm overflow-x-auto py-2">
      {crumbs.map((crumb, idx) => {
        const isLast = idx === crumbs.length - 1;
        return (
          <React.Fragment key={crumb.path}>
            {idx > 0 && <ChevronRightIcon className="w-3.5 h-3.5 text-charcoal-400 flex-shrink-0" />}
            <button
              type="button"
              onClick={() => onNavigate(crumb.path)}
              disabled={isLast}
              className={`flex items-center space-x-1.5 px-2 py-1 rounded text-sm transition-colors flex-shrink-0 ${
                isLast
                  ? 'font-medium text-charcoal-900 cursor-default bg-charcoal-100/50'
                  : 'text-charcoal-600 hover:text-charcoal-900 hover:bg-charcoal-100'
              }`}
              title={crumb.name}
            >
              {idx === 0 && <FolderIcon className="w-4 h-4 text-accent flex-shrink-0" />}
              <span>{crumb.name}</span>
            </button>
          </React.Fragment>
        );
      })}
    </nav>
  );
};
