import React from 'react';
import { FolderIcon, ShareIcon, PlusIcon, UploadIcon, CloseIcon, TrashIcon } from './Icons';

interface SidebarProps {
  activeTab: 'files' | 'shares' | 'trash';
  onSelectTab: (tab: 'files' | 'shares' | 'trash') => void;
  onNewFolder: () => void;
  onUpload: () => void;
  isOpenMobile: boolean;
  onCloseMobile: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  activeTab,
  onSelectTab,
  onNewFolder,
  onUpload,
  isOpenMobile,
  onCloseMobile,
}) => {
  const content = (
    <div className="flex flex-col h-full bg-white border-r border-charcoal-200 w-60 p-3 select-none">
      <div className="flex items-center justify-between pb-3 md:hidden">
        <span className="font-semibold text-charcoal-900">Menü</span>
        <button
          type="button"
          onClick={onCloseMobile}
          className="p-1 text-charcoal-500 hover:text-charcoal-800 rounded"
          aria-label="Kapat"
        >
          <CloseIcon className="w-5 h-5" />
        </button>
      </div>

      {/* Action buttons */}
      <div className="space-y-2 mb-4">
        <button
          type="button"
          onClick={() => {
            onUpload();
            onCloseMobile();
          }}
          className="w-full flex items-center justify-center space-x-2 py-2 px-3 bg-accent hover:bg-accent-hover text-white text-sm font-medium rounded-lg shadow-sm transition-colors"
        >
          <UploadIcon className="w-4 h-4" />
          <span>Dosya Yükle</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onNewFolder();
            onCloseMobile();
          }}
          className="w-full flex items-center justify-center space-x-2 py-2 px-3 bg-warm-100 hover:bg-warm-200 text-charcoal-800 border border-charcoal-200 text-sm font-medium rounded-lg transition-colors"
        >
          <PlusIcon className="w-4 h-4 text-charcoal-600" />
          <span>Yeni Klasör</span>
        </button>
      </div>

      {/* Navigation links */}
      <nav className="flex-1 space-y-1">
        <button
          type="button"
          onClick={() => {
            onSelectTab('files');
            onCloseMobile();
          }}
          className={`w-full flex items-center space-x-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'files'
              ? 'bg-accent/10 text-accent font-semibold'
              : 'text-charcoal-700 hover:bg-charcoal-100 hover:text-charcoal-900'
          }`}
        >
          <FolderIcon className={`w-4 h-4 ${activeTab === 'files' ? 'text-accent' : 'text-charcoal-500'}`} />
          <span>Dosyalar</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onSelectTab('shares');
            onCloseMobile();
          }}
          className={`w-full flex items-center space-x-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'shares'
              ? 'bg-accent/10 text-accent font-semibold'
              : 'text-charcoal-700 hover:bg-charcoal-100 hover:text-charcoal-900'
          }`}
        >
          <ShareIcon className={`w-4 h-4 ${activeTab === 'shares' ? 'text-accent' : 'text-charcoal-500'}`} />
          <span>Paylaşımlar</span>
        </button>

        <button
          type="button"
          onClick={() => {
            onSelectTab('trash');
            onCloseMobile();
          }}
          className={`w-full flex items-center space-x-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
            activeTab === 'trash'
              ? 'bg-accent/10 text-accent font-semibold'
              : 'text-charcoal-700 hover:bg-charcoal-100 hover:text-charcoal-900'
          }`}
        >
          <TrashIcon className={`w-4 h-4 ${activeTab === 'trash' ? 'text-accent' : 'text-charcoal-500'}`} />
          <span>Çöp Kutusu</span>
        </button>
      </nav>

      {/* Footer info */}
      <div className="pt-3 border-t border-charcoal-200 text-xs text-charcoal-400">
        <div className="flex items-center justify-between">
          <span>Depgo Yerel Depolama</span>
          <span className="w-2 h-2 rounded-full bg-accent" title="Aktif" />
        </div>
      </div>
    </div>
  );

  return (
    <>
      {/* Desktop sidebar */}
      <aside className="hidden md:block flex-shrink-0">
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
