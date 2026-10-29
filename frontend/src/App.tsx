import React, { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { User, FileItem } from './types';
import { authApi, fsApi, setOnAuthFailure } from './api/client';
import { joinPath } from './utils/format';
import { Navbar } from './components/Navbar';
import { Sidebar } from './components/Sidebar';
import { Breadcrumbs } from './components/Breadcrumbs';
import { FilterInput } from './components/FilterInput';
import { FileList } from './components/FileList';
import { FileGrid } from './components/FileGrid';
import { NewFolderModal } from './components/NewFolderModal';
import { UploadModal } from './components/UploadModal';
import { RenameModal } from './components/RenameModal';
import { MoveModal } from './components/MoveModal';
import { DeleteDialog } from './components/DeleteDialog';
import { ShareModal } from './components/ShareModal';
import { PreviewModal } from './components/PreviewModal';
import { ShareManagement } from './components/ShareManagement';
import { TrashManagement } from './components/TrashManagement';
import { TextEditorModal } from './components/TextEditorModal';
import { VersionHistoryModal } from './components/VersionHistoryModal';
import { GuestShareView } from './components/GuestShareView';
import { LoginView } from './components/LoginView';
import { SelectionBar } from './components/SelectionBar';
import { ContextMenu, ContextMenuPosition } from './components/ContextMenu';
import { DragPreview } from './components/DragPreview';
import { RefreshIcon, SpinnerIcon, PlusIcon, UploadIcon, GridIcon, ListIcon } from './components/Icons';
import { describeError } from './utils/errors';
import { useTranslation } from './utils/i18n';

export const App: React.FC = () => {
  const { t } = useTranslation();
  // Check if current URL is a guest share (/s/:share_id)
  const [currentUrlPath, setCurrentUrlPath] = useState(window.location.pathname);

  useEffect(() => {
    const handlePopState = () => {
      setCurrentUrlPath(window.location.pathname);
    };
    window.addEventListener('popstate', handlePopState);
    return () => window.removeEventListener('popstate', handlePopState);
  }, []);

  const guestShareId = useMemo(() => {
    if (currentUrlPath.startsWith('/s/')) {
      const match = currentUrlPath.replace(/^\/s\//, '').split('/')[0];
      return match || null;
    }
    return null;
  }, [currentUrlPath]);

  // Auth state
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(authApi.isAuthenticated());
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    setOnAuthFailure(() => {
      setIsAuthenticated(false);
      setUser(null);
    });
    return () => setOnAuthFailure(null);
  }, []);

  useEffect(() => {
    if (isAuthenticated && !guestShareId) {
      authApi
        .me()
        .then(setUser)
        .catch(() => {
          setIsAuthenticated(false);
          setUser(null);
        });
    }
  }, [isAuthenticated, guestShareId]);

  // Main UI state
  const [activeTab, setActiveTab] = useState<'files' | 'shares' | 'trash'>('files');
  const [currentDir, setCurrentDir] = useState<string>('/');
  const [items, setItems] = useState<FileItem[]>([]);
  const [totalFiles, setTotalFiles] = useState(0);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const perPage = 100;

  const [filterQuery, setFilterQuery] = useState('');
  const [viewMode, setViewMode] = useState<'list' | 'grid'>('list');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  // Modals state
  const [newFolderOpen, setNewFolderOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [renameTarget, setRenameTarget] = useState<FileItem | null>(null);
  const [moveTarget, setMoveTarget] = useState<FileItem | null>(null);
  const [batchMoveTargets, setBatchMoveTargets] = useState<FileItem[] | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<FileItem | null>(null);
  const [batchDeleteTargets, setBatchDeleteTargets] = useState<FileItem[] | null>(null);
  const [shareTarget, setShareTarget] = useState<FileItem | null>(null);
  const [previewTarget, setPreviewTarget] = useState<{ item: FileItem; rawUrl: string } | null>(null);
  const [textEditorTarget, setTextEditorTarget] = useState<{ filePath: string; fileName: string } | null>(null);
  const [versionHistoryTarget, setVersionHistoryTarget] = useState<{ filePath: string; fileName: string } | null>(null);

  // Selection & Context Menu state
  const [selectedNames, setSelectedNames] = useState<Set<string>>(new Set());
  const [contextMenu, setContextMenu] = useState<ContextMenuPosition | null>(null);

  // Drag and drop state
  const [draggedItems, setDraggedItems] = useState<FileItem[]>([]);
  const [dragOverItemName, setDragOverItemName] = useState<string | null>(null);
  const [dragOverCrumbPath, setDragOverCrumbPath] = useState<string | null>(null);
  const dragPreviewRef = useRef<HTMLDivElement>(null);

  // Fetch file list with explicit pagination
  const loadFiles = useCallback(
    async (path = currentDir, targetPage = 1, refresh = false) => {
      try {
        setLoading(true);
        setError(null);
        const resp = await fsApi.list(path, '', refresh, targetPage, perPage);
        setItems(resp.content || []);
        setTotalFiles(resp.total || 0);
        setHasMore(!!resp.has_more);
        setPage(targetPage);
        setCurrentDir(path);
        setSelectedNames(new Set());
      } catch (err: unknown) {
        setError(describeError(err, 'Dosyalar yüklenemedi'));
      } finally {
        setLoading(false);
      }
    },
    [currentDir]
  );

  useEffect(() => {
    if (isAuthenticated && !guestShareId && activeTab === 'files') {
      loadFiles(currentDir, 1);
    }
  }, [isAuthenticated, guestShareId, activeTab, currentDir, loadFiles]);

  // Filtered files
  const filteredItems = useMemo(() => {
    if (!filterQuery.trim()) return items;
    const lower = filterQuery.toLowerCase();
    return items.filter((item) => item.name.toLowerCase().includes(lower));
  }, [items, filterQuery]);

  // Handlers for file operations
  const handleOpenFolder = (folder: FileItem) => {
    const nextPath = joinPath(currentDir, folder.name);
    setFilterQuery('');
    setPage(1);
    loadFiles(nextPath, 1);
  };

  const handleNavigateBreadcrumb = (path: string) => {
    setFilterQuery('');
    setPage(1);
    loadFiles(path, 1);
  };

  const handlePreview = async (item: FileItem) => {
    try {
      const fullPath = joinPath(currentDir, item.name);
      const fileData = await fsApi.get(fullPath);
      setPreviewTarget({
        item,
        rawUrl: fileData.raw_url,
      });
    } catch (err: unknown) {
      alert(describeError(err, 'Önizleme bağlantısı alınamadı'));
    }
  };

  const handleEdit = (item: FileItem) => {
    const fullPath = joinPath(currentDir, item.name);
    setTextEditorTarget({
      filePath: fullPath,
      fileName: item.name,
    });
  };

  const handleVersions = (item: FileItem) => {
    const fullPath = joinPath(currentDir, item.name);
    setVersionHistoryTarget({
      filePath: fullPath,
      fileName: item.name,
    });
  };

  const handleDownload = async (item: FileItem) => {
    try {
      const fullPath = joinPath(currentDir, item.name);
      const fileData = await fsApi.get(fullPath);
      if (fileData.raw_url) {
        window.open(fileData.raw_url, '_blank');
      }
    } catch (err: unknown) {
      alert(describeError(err, 'İndirme bağlantısı alınamadı'));
    }
  };

  const handleNewFolderSubmit = async (folderName: string) => {
    const fullPath = joinPath(currentDir, folderName);
    await fsApi.mkdir(fullPath);
    await loadFiles(currentDir, page, true);
  };

  const handleRenameSubmit = async (item: FileItem, newName: string) => {
    const fullPath = joinPath(currentDir, item.name);
    await fsApi.rename(fullPath, newName);
    await loadFiles(currentDir, page, true);
  };

  const handleMoveSubmit = async (targetItems: FileItem[], targetDir: string) => {
    const names = targetItems.map((i) => i.name);
    await fsApi.move(currentDir, targetDir, names);
    await loadFiles(currentDir, page, true);
  };

  const handleDeleteConfirm = async (targetItems: FileItem[]) => {
    const names = targetItems.map((i) => i.name);
    await fsApi.remove(currentDir, names);
    await loadFiles(currentDir, page, true);
  };

  // Selection handlers
  const handleToggleSelect = (name: string) => {
    setSelectedNames((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  };

  const handleToggleSelectAll = () => {
    setSelectedNames((prev) => {
      if (prev.size === filteredItems.length && filteredItems.length > 0) {
        return new Set();
      }
      return new Set(filteredItems.map((i) => i.name));
    });
  };

  const handleClearSelection = () => {
    setSelectedNames(new Set());
  };

  const selectedItems = useMemo(() => {
    return items.filter((item) => selectedNames.has(item.name));
  }, [items, selectedNames]);

  const handleBatchDownload = async () => {
    for (const item of selectedItems) {
      if (!item.is_dir) {
        await handleDownload(item);
      }
    }
  };

  const handleContextMenu = (item: FileItem, e: React.MouseEvent) => {
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      item,
    });
  };

  // Drag and drop handlers
  const handleDragStartItem = (item: FileItem, e: React.DragEvent) => {
    const targets = selectedNames.has(item.name)
      ? items.filter((i) => selectedNames.has(i.name))
      : [item];
    setDraggedItems(targets);
    e.dataTransfer.setData('text/plain', targets.map((i) => i.name).join(','));
    e.dataTransfer.effectAllowed = 'move';
    if (dragPreviewRef.current) {
      e.dataTransfer.setDragImage(dragPreviewRef.current, 10, 10);
    }
  };

  const handleDragOverFolder = (folder: FileItem, e: React.DragEvent) => {
    if (draggedItems.some((i) => i.name === folder.name)) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
    setDragOverItemName(folder.name);
  };

  const handleDragLeaveFolder = (folder: FileItem) => {
    if (dragOverItemName === folder.name) {
      setDragOverItemName(null);
    }
  };

  const handleDropToFolder = async (folder: FileItem, e: React.DragEvent) => {
    e.preventDefault();
    setDragOverItemName(null);
    const targets = draggedItems.filter((i) => i.name !== folder.name);
    if (targets.length === 0) return;
    const targetDir = joinPath(currentDir, folder.name);
    try {
      setLoading(true);
      setError(null);
      await fsApi.move(currentDir, targetDir, targets.map((i) => i.name));
      await loadFiles(currentDir, page, true);
      setSelectedNames(new Set());
    } catch (err: unknown) {
      setError(describeError(err, t('files.move_error')));
    } finally {
      setLoading(false);
      setDraggedItems([]);
    }
  };

  const handleDropToBreadcrumb = async (targetPath: string, e: React.DragEvent) => {
    e.preventDefault();
    setDragOverCrumbPath(null);
    if (targetPath === currentDir || draggedItems.length === 0) return;
    try {
      setLoading(true);
      setError(null);
      await fsApi.move(currentDir, targetPath, draggedItems.map((i) => i.name));
      await loadFiles(currentDir, page, true);
      setSelectedNames(new Set());
    } catch (err: unknown) {
      setError(describeError(err, t('files.move_error')));
    } finally {
      setLoading(false);
      setDraggedItems([]);
    }
  };

  const handleLogout = async () => {
    await authApi.logout();
    setIsAuthenticated(false);
    setUser(null);
  };

  const totalPages = Math.ceil(totalFiles / perPage) || 1;

  // If Guest Share route /s/:share_id, render GuestShareView
  if (guestShareId) {
    return <GuestShareView shareId={guestShareId} />;
  }

  // If not authenticated, render LoginView
  if (!isAuthenticated) {
    return <LoginView onLoginSuccess={() => setIsAuthenticated(true)} />;
  }

  return (
    <div className="min-h-screen bg-warm-50 flex flex-col">
      <Navbar
        user={user}
        viewMode={viewMode}
        onToggleViewMode={() => setViewMode((prev) => (prev === 'list' ? 'grid' : 'list'))}
        onLogout={handleLogout}
        onOpenMobileMenu={() => setMobileMenuOpen(true)}
      />

      <div className="flex-1 flex overflow-hidden">
        <Sidebar
          activeTab={activeTab}
          onSelectTab={(tab) => {
            setActiveTab(tab);
            setFilterQuery('');
          }}
          isOpenMobile={mobileMenuOpen}
          onCloseMobile={() => setMobileMenuOpen(false)}
        />

        <main className="flex-1 overflow-y-auto p-4 sm:p-6">
          <div className="max-w-7xl mx-auto space-y-4">
            {activeTab === 'files' ? (
              <>
                {/* Nextcloud Files: Breadcrumbs and Toolbar */}
                <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 bg-white border border-charcoal-200 rounded-xl px-4 py-2.5 shadow-xs">
                  <div className="min-w-0 flex-1">
                    <Breadcrumbs
                      currentPath={currentDir}
                      onNavigate={handleNavigateBreadcrumb}
                      onDropToPath={handleDropToBreadcrumb}
                      dragOverPath={dragOverCrumbPath}
                    />
                  </div>

                  <div className="flex items-center flex-wrap gap-2 flex-shrink-0">
                    <button
                      type="button"
                      onClick={() => setNewFolderOpen(true)}
                      className="flex items-center space-x-1.5 px-3 py-1.5 bg-warm-100 hover:bg-warm-200 text-charcoal-800 border border-charcoal-200 text-xs font-semibold rounded-lg transition-colors"
                      title={t('files.new_folder')}
                    >
                      <PlusIcon className="w-4 h-4 text-charcoal-600" />
                      <span>{t('files.new_folder')}</span>
                    </button>

                    <button
                      type="button"
                      onClick={() => setUploadOpen(true)}
                      className="flex items-center space-x-1.5 px-3 py-1.5 bg-accent hover:bg-accent-hover text-white text-xs font-semibold rounded-lg shadow-xs transition-colors"
                      title={t('files.upload')}
                    >
                      <UploadIcon className="w-4 h-4" />
                      <span>{t('files.upload')}</span>
                    </button>

                    <div className="h-5 w-px bg-charcoal-200 mx-0.5 hidden sm:block" />

                    <button
                      type="button"
                      onClick={() => setViewMode((prev) => (prev === 'list' ? 'grid' : 'list'))}
                      className="p-1.5 text-charcoal-600 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-lg transition-colors"
                      title={viewMode === 'list' ? t('nav.view_grid') : t('nav.view_list')}
                      aria-label={viewMode === 'list' ? t('nav.view_grid') : t('nav.view_list')}
                    >
                      {viewMode === 'list' ? <GridIcon className="w-4 h-4" /> : <ListIcon className="w-4 h-4" />}
                    </button>

                    <FilterInput value={filterQuery} onChange={setFilterQuery} />

                    <button
                      type="button"
                      onClick={() => loadFiles(currentDir, page, true)}
                      disabled={loading}
                      className="p-1.5 text-charcoal-600 hover:text-charcoal-900 hover:bg-charcoal-100 rounded-lg transition-colors flex-shrink-0"
                      title={t('common.refresh')}
                    >
                      <RefreshIcon className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
                    </button>
                  </div>
                </div>

                {/* Nextcloud Selection Bar */}
                {selectedItems.length > 0 && (
                  <SelectionBar
                    selectedItems={selectedItems}
                    onClearSelection={handleClearSelection}
                    onBatchDownload={handleBatchDownload}
                    onBatchMove={() => setBatchMoveTargets(selectedItems)}
                    onBatchDelete={() => setBatchDeleteTargets(selectedItems)}
                  />
                )}

                {error && (
                  <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-xs rounded-xl">
                    {error}
                  </div>
                )}

                {loading && items.length === 0 ? (
                  <div className="py-20 text-center text-charcoal-500">
                    <SpinnerIcon className="w-8 h-8 text-accent animate-spin mx-auto mb-2" />
                    <span className="text-sm">Dosyalar yükleniyor...</span>
                  </div>
                ) : viewMode === 'list' ? (
                  <FileList
                    items={filteredItems}
                    selectedNames={selectedNames}
                    onToggleSelect={handleToggleSelect}
                    onToggleSelectAll={handleToggleSelectAll}
                    onOpenFolder={handleOpenFolder}
                    onPreview={handlePreview}
                    onEdit={handleEdit}
                    onVersions={handleVersions}
                    onDownload={handleDownload}
                    onShare={(item) => setShareTarget(item)}
                    onRename={(item) => setRenameTarget(item)}
                    onMove={(item) => setMoveTarget(item)}
                    onDelete={(item) => setDeleteTarget(item)}
                    onContextMenu={handleContextMenu}
                    onDragStartItem={handleDragStartItem}
                    onDragOverFolder={handleDragOverFolder}
                    onDragLeaveFolder={handleDragLeaveFolder}
                    onDropToFolder={handleDropToFolder}
                    dragOverItemName={dragOverItemName}
                  />
                ) : (
                  <FileGrid
                    items={filteredItems}
                    selectedNames={selectedNames}
                    onToggleSelect={handleToggleSelect}
                    onOpenFolder={handleOpenFolder}
                    onPreview={handlePreview}
                    onEdit={handleEdit}
                    onVersions={handleVersions}
                    onDownload={handleDownload}
                    onShare={(item) => setShareTarget(item)}
                    onRename={(item) => setRenameTarget(item)}
                    onMove={(item) => setMoveTarget(item)}
                    onDelete={(item) => setDeleteTarget(item)}
                    onContextMenu={handleContextMenu}
                    onDragStartItem={handleDragStartItem}
                    onDragOverFolder={handleDragOverFolder}
                    onDragLeaveFolder={handleDragLeaveFolder}
                    onDropToFolder={handleDropToFolder}
                    dragOverItemName={dragOverItemName}
                  />
                )}

                {/* File list pagination bar (prevent silent truncation) */}
                {(totalFiles > perPage || hasMore) && !filterQuery && (
                  <div className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 bg-white border border-charcoal-200 rounded-xl text-xs text-charcoal-600 shadow-xs">
                    <span>
                      {t('files.pagination_summary', {
                        total: totalFiles,
                        start: (page - 1) * perPage + 1,
                        end: Math.min(page * perPage, totalFiles),
                      })}
                    </span>
                    <div className="flex items-center space-x-2">
                      <button
                        type="button"
                        onClick={() => loadFiles(currentDir, page - 1)}
                        disabled={page <= 1 || loading}
                        className="px-2.5 py-1 bg-white border border-charcoal-200 rounded-md hover:bg-warm-100 disabled:opacity-40 transition-colors"
                      >
                        {t('files.prev_page')}
                      </button>
                      <span className="font-semibold text-charcoal-800">
                        {page} / {totalPages}
                      </span>
                      <button
                        type="button"
                        onClick={() => loadFiles(currentDir, page + 1)}
                        disabled={(!hasMore && page >= totalPages) || loading}
                        className="px-2.5 py-1 bg-white border border-charcoal-200 rounded-md hover:bg-warm-100 disabled:opacity-40 transition-colors"
                      >
                        {t('files.next_page')}
                      </button>
                    </div>
                  </div>
                )}
              </>
            ) : activeTab === 'shares' ? (
              <ShareManagement />
            ) : (
              <TrashManagement />
            )}
          </div>
        </main>
      </div>

      {/* Modals */}
      <NewFolderModal
        isOpen={newFolderOpen}
        onClose={() => setNewFolderOpen(false)}
        onSubmit={handleNewFolderSubmit}
      />

      <UploadModal
        isOpen={uploadOpen}
        currentDir={currentDir}
        onClose={() => setUploadOpen(false)}
        onCompleted={() => loadFiles(currentDir, page, true)}
      />

      <RenameModal
        isOpen={!!renameTarget}
        item={renameTarget}
        onClose={() => setRenameTarget(null)}
        onSubmit={handleRenameSubmit}
      />

      <MoveModal
        isOpen={!!moveTarget || (!!batchMoveTargets && batchMoveTargets.length > 0)}
        item={moveTarget}
        items={batchMoveTargets || (moveTarget ? [moveTarget] : [])}
        currentDir={currentDir}
        onClose={() => {
          setMoveTarget(null);
          setBatchMoveTargets(null);
        }}
        onSubmit={handleMoveSubmit}
      />

      <DeleteDialog
        isOpen={!!deleteTarget || (!!batchDeleteTargets && batchDeleteTargets.length > 0)}
        item={deleteTarget}
        items={batchDeleteTargets || (deleteTarget ? [deleteTarget] : [])}
        onClose={() => {
          setDeleteTarget(null);
          setBatchDeleteTargets(null);
        }}
        onConfirm={handleDeleteConfirm}
      />

      <ContextMenu
        position={contextMenu}
        onClose={() => setContextMenu(null)}
        onPreview={handlePreview}
        onEdit={handleEdit}
        onVersions={handleVersions}
        onDownload={handleDownload}
        onShare={(item) => setShareTarget(item)}
        onRename={(item) => setRenameTarget(item)}
        onMove={(item) => setMoveTarget(item)}
        onDelete={(item) => setDeleteTarget(item)}
      />

      <DragPreview
        ref={dragPreviewRef}
        names={draggedItems.map((i) => i.name)}
        isFolder={draggedItems.length === 1 && draggedItems[0].is_dir}
      />

      <ShareModal
        isOpen={!!shareTarget}
        item={shareTarget}
        onClose={() => setShareTarget(null)}
      />

      {previewTarget && (
        <PreviewModal
          isOpen={true}
          name={previewTarget.item.name}
          size={previewTarget.item.size}
          rawUrl={previewTarget.rawUrl}
          onClose={() => setPreviewTarget(null)}
          onDownload={() => handleDownload(previewTarget.item)}
        />
      )}

      {textEditorTarget && (
        <TextEditorModal
          isOpen={true}
          filePath={textEditorTarget.filePath}
          fileName={textEditorTarget.fileName}
          onClose={() => setTextEditorTarget(null)}
          onSaved={() => loadFiles(currentDir, page, true)}
        />
      )}

      {versionHistoryTarget && (
        <VersionHistoryModal
          isOpen={true}
          filePath={versionHistoryTarget.filePath}
          fileName={versionHistoryTarget.fileName}
          onClose={() => setVersionHistoryTarget(null)}
          onRestored={() => loadFiles(currentDir, page, true)}
        />
      )}
    </div>
  );
};
