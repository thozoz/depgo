import { useState, useEffect } from 'react';

export type Language = 'tr' | 'en';

const STORAGE_KEY = 'depgo_language';

function getStoredLanguage(): Language {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'en' || saved === 'tr') {
      return saved;
    }
  } catch {
    // localStorage may be disabled or restricted
  }
  return 'tr';
}

let currentLanguage: Language = getStoredLanguage();
const listeners = new Set<(lang: Language) => void>();

export function getLanguage(): Language {
  return currentLanguage;
}

export function setLanguage(lang: Language): void {
  if (lang !== 'tr' && lang !== 'en') return;
  currentLanguage = lang;
  try {
    localStorage.setItem(STORAGE_KEY, lang);
  } catch {
    // ignore storage quota / disabled errors
  }
  listeners.forEach((fn) => fn(lang));
}

export function subscribeLanguage(fn: (lang: Language) => void): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

export const translations: Record<Language, Record<string, string>> = {
  tr: {
    // Genel
    'app.name': 'Depgo',
    'app.tagline': 'Hafif ve Hızlı Bulut Depolama',
    'common.close': 'Kapat',
    'common.cancel': 'İptal',
    'common.save': 'Kaydet',
    'common.saving': 'Kaydediliyor...',
    'common.refresh': 'Yenile',
    'common.loading': 'Yükleniyor...',
    'common.error': 'Hata',
    'common.success': 'Başarılı',
    'common.actions': 'İşlemler',
    'common.back': 'Geri',
    'common.delete': 'Sil',
    'common.rename': 'Yeniden Adlandır',
    'common.download': 'İndir',
    'common.share': 'Paylaş',
    'common.edit': 'Metni Düzenle',
    'common.move': 'Taşı',
    'common.preview': 'Önizle',
    'common.versions': 'Sürüm Geçmişi',
    'common.restore': 'Geri Yükle',
    'common.permanent_delete': 'Kalıcı Olarak Sil',
    'common.yes_permanent_delete': 'Evet, Kalıcı Olarak Sil',
    'common.menu': 'Menü',
    'common.home': 'Ana Sayfaya Dön',
    'common.copy': 'Kopyala',
    'common.copied': 'Kopyalandı',

    // Nav & Sidebar
    'nav.files': 'Dosyalar',
    'nav.shares': 'Paylaşımlar',
    'nav.trash': 'Çöp Kutusu',
    'nav.logout': 'Çıkış Yap',
    'nav.switch_lang': 'Dil Seçimi',
    'nav.view_list': 'Liste Görünümü',
    'nav.view_grid': 'Izgara Görünümü',
    'nav.storage_quota': 'Depolama',
    'nav.open_menu': 'Menüyü aç',

    // Giriş (Login)
    'login.title': 'Depgo Girişi',
    'login.subtitle': 'Devam etmek için yönetici kimliğinizi doğrulayın.',
    'login.username_label': 'Kullanıcı Adı',
    'login.password_label': 'Şifre',
    'login.submit_button': 'Giriş Yap',
    'login.submitting': 'Giriş yapılıyor...',

    // Dosya Gezgini & Araç Çubuğu
    'files.new_folder': 'Yeni Klasör',
    'files.upload': 'Dosya Yükle',
    'files.upload_start': 'Yüklemeyi Başlat',
    'files.filter_placeholder': 'Dosya veya klasör ara...',
    'files.name': 'Ad',
    'files.size': 'Boyut',
    'files.modified': 'Değiştirilme',
    'files.empty_directory': 'Bu klasörde görüntülenecek öğe bulunamadı.',
    'files.loading_files': 'Dosyalar yükleniyor...',
    'files.load_failed': 'Dosyalar yüklenemedi',
    'files.selected_count': '{count} öğe seçildi',
    'files.select_all': 'Tümünü Seç',
    'files.select_item': '{name} seç',
    'files.actions_menu': '{name} için işlemler menüsü',
    'files.move_success': 'Öğeler başarıyla taşındı',
    'files.move_conflict': 'Hedef klasörde aynı isimde dosya veya klasör zaten var (409).',
    'files.move_error': 'Taşıma işlemi başarısız oldu',
    'files.pagination_summary': 'Toplam {total} öğeden {start} - {end} arası gösteriliyor',
    'files.prev_page': 'Önceki Sayfa',
    'files.next_page': 'Sonraki Sayfa',
    'files.previous': 'Önceki',
    'files.next': 'Sonraki',

    // Yeni Klasör Modalı
    'folder.create_title': 'Yeni Klasör Oluştur',
    'folder.name_label': 'Klasör Adı',
    'folder.placeholder': 'Örn. Projeler',
    'folder.empty_error': 'Klasör adı boş bırakılamaz',
    'folder.invalid_chars': 'Klasör adı / veya \\ içeremez',
    'folder.create_button': 'Oluştur',
    'folder.creating': 'Oluşturuluyor...',

    // Yükleme Modalı
    'upload.title': 'Dosya Yükle',
    'upload.dropzone_text': 'Dosyaları buraya sürükleyip bırakın veya seçmek için tıklayın',
    'upload.overwrite': 'Üzerine Yaz',
    'upload.conflict_warning': 'Bu konumda aynı isimde bir dosya zaten var.',
    'upload.remove_from_list': 'Listeden çıkar',
    'upload.no_files_selected': 'Dosya seçilmedi',
    'upload.files_selected': '{count} dosya seçildi',
    'upload.uploading': 'Yükleniyor...',

    // Yeniden Adlandır Modalı
    'rename.title': 'Yeniden Adlandır',
    'rename.name_label': 'Yeni Ad',
    'rename.empty_error': 'Ad boş bırakılamaz',
    'rename.invalid_chars': 'Ad / veya \\ içeremez',
    'rename.save_button': 'Kaydet',

    // Taşıma Modalı
    'move.title': 'Öğeyi Taşı',
    'move.moving_item': 'Taşınacak:',
    'move.target_dir_label': 'Hedef Klasör Yolu',
    'move.example_path': 'Örnek: /belgeler veya /',
    'move.same_folder_error': 'Hedef klasör mevcut klasör ile aynı olamaz',
    'move.submit_button': 'Taşı',

    // Silme Diyaloğu
    'delete.title': 'Öğeyi Sil',
    'delete.single_confirm': '{name} {type} silmek istediğinize emin misiniz?',
    'delete.multiple_confirm': 'Seçili {count} öğeyi silmek istediğinize emin misiniz?',
    'delete.folder': 'klasörünü',
    'delete.file': 'dosyasını',
    'delete.submit_button': 'Sil',

    // Paylaşım & Misafir
    'share.title': 'Paylaşım Bağlantısı Oluştur',
    'share.ready_title': 'Paylaşım Bağlantınız Hazır',
    'share.slug_label': 'Özel Bağlantı Adı (Slug)',
    'share.password_label': 'Erişim Parolası (İsteğe bağlı)',
    'share.burn_after_read': 'Okunduktan sonra bu paylaşımı sil',
    'share.burn_description': 'Bağlantı bir kez açılıp içeriği alınınca paylaşım silinir. Dosyanın kendisi silinmez.',
    'share.submit': 'Paylaşım Oluştur',
    'share.active': 'Aktif',
    'share.disabled': 'Devre Dışı',
    'share.enable': 'Etkinleştir',
    'share.disable': 'Kapat',
    'share.guest_title': 'Depgo Paylaşımı',
    'share.guest_burn_notice': 'Bu bağlantı tek okumalık: dosyayı önizlediğinizde veya indirdiğinizde paylaşım silinir ve bir daha açılamaz.',
    'share.unlock_button': 'Erişimi Aç',
    'share.password_placeholder': 'Parola',
    'share.protected_title': 'Korumalı Paylaşım',
    'share.protected_desc': 'Bu içeriğe erişmek için lütfen erişim parolasını girin.',
    'share.verifying': 'Doğrulanıyor...',
    'share.no_active': 'Henüz aktif bir paylaşım bağlantısı yok',
    'share.no_active_desc': 'Dosyalar sekmesinden bir dosya veya klasör seçip "Paylaş" butonuna tıklayarak bağlantı oluşturabilirsiniz.',
    'share.link_copied': 'Bağlantı panoya kopyalandı',
    'share.copy_button': 'Bağlantıyı Kopyala',

    // Çöp Kutusu (Trash)
    'trash.title': 'Çöp Kutusu',
    'trash.subtitle': 'Silinmiş yerel dosya ve klasörleri görüntüleyin, geri yükleyin veya kalıcı olarak temizleyin.',
    'trash.empty': 'Çöp kutusu boş',
    'trash.empty_desc': 'Silinen öğeler burada saklanır.',
    'trash.original_path': 'Orijinal Yol & Ad',
    'trash.deleted_at': 'Silinme Tarihi',
    'trash.restore': 'Geri Yükle',
    'trash.permanent_delete': 'Kalıcı Olarak Sil',
    'trash.collision_title': 'Geri Yükleme Çakışması',
    'trash.collision_desc': 'Orijinal konumda ({path}) aynı isimde bir öğe zaten mevcut.',
    'trash.custom_path_label': 'Alternatif Geri Yükleme Yolu',
    'trash.restore_to_path': 'Bu Konuma Geri Yükle',
    'trash.permanent_delete_title': 'Kalıcı Olarak Sil',
    'trash.permanent_delete_confirm': '{path} öğesini kalıcı olarak silmek istediğinize emin misiniz? Bu işlem geri alınamaz.',

    // Metin Editörü & Sürüm Geçmişi
    'editor.content_label': 'Metin düzenleyici içeriği',
    'editor.modified_badge': 'Değiştirildi',
    'editor.save': 'Kaydet',
    'editor.unsaved_warning': 'Kaydedilmemiş değişiklikleriniz var. Kapatmak istediğinize emin misiniz?',
    'versions.title': 'Sürüm Geçmişi',
    'versions.restore_confirm': 'Revizyon #{revision} sürümüne dönmek istediğinize emin misiniz? Mevcut içerik geçmişe kaydedilerek korunacaktır.',
    'versions.restore_button': 'Geri Dön',
    'preview.preparing': 'Önizleme hazırlanıyor...',

    // Hata Mesajları
    'error.incorrect_credentials': 'Kullanıcı adı veya parola hatalı.',
    'error.incorrect_password': 'Parola hatalı.',
    'error.share_already_exists': 'Bu bağlantı adı zaten kullanılıyor. Başka bir ad seçin.',
    'error.too_many_attempts': 'Çok fazla hatalı deneme yapıldı. Lütfen biraz sonra tekrar deneyin.',
    'error.share_not_found': 'Bu paylaşım bulunamadı. Silinmiş ya da kapatılmış olabilir; tek okumalık bir bağlantı ise zaten kullanılmış olabilir.',
    'error.share_expired': 'Bu paylaşımın süresi dolmuş veya kullanım hakkı bitmiş.',
    'error.unexpected': 'Beklenmeyen bir hata oluştu.',
    'error.conflict': 'Aynı ada sahip bir öğe zaten mevcut.',
  },
  en: {
    // General
    'app.name': 'Depgo',
    'app.tagline': 'Lightweight and Fast Cloud Storage',
    'common.close': 'Close',
    'common.cancel': 'Cancel',
    'common.save': 'Save',
    'common.saving': 'Saving...',
    'common.refresh': 'Refresh',
    'common.loading': 'Loading...',
    'common.error': 'Error',
    'common.success': 'Success',
    'common.actions': 'Actions',
    'common.back': 'Back',
    'common.delete': 'Delete',
    'common.rename': 'Rename',
    'common.download': 'Download',
    'common.share': 'Share',
    'common.edit': 'Edit Text',
    'common.move': 'Move',
    'common.preview': 'Preview',
    'common.versions': 'Version History',
    'common.restore': 'Restore',
    'common.permanent_delete': 'Delete Permanently',
    'common.yes_permanent_delete': 'Yes, Delete Permanently',
    'common.menu': 'Menu',
    'common.home': 'Back to Home',
    'common.copy': 'Copy',
    'common.copied': 'Copied',

    // Nav & Sidebar
    'nav.files': 'Files',
    'nav.shares': 'Shares',
    'nav.trash': 'Trash',
    'nav.logout': 'Sign Out',
    'nav.switch_lang': 'Language',
    'nav.view_list': 'List View',
    'nav.view_grid': 'Grid View',
    'nav.storage_quota': 'Storage',
    'nav.open_menu': 'Open menu',

    // Login
    'login.title': 'Depgo Sign In',
    'login.subtitle': 'Authenticate your administrator identity to continue.',
    'login.username_label': 'Username',
    'login.password_label': 'Password',
    'login.submit_button': 'Sign In',
    'login.submitting': 'Signing in...',

    // Files Explorer & Toolbar
    'files.new_folder': 'New Folder',
    'files.upload': 'Upload File',
    'files.upload_start': 'Start Upload',
    'files.filter_placeholder': 'Search files or folders...',
    'files.name': 'Name',
    'files.size': 'Size',
    'files.modified': 'Modified',
    'files.empty_directory': 'No items to display in this folder.',
    'files.loading_files': 'Loading files...',
    'files.load_failed': 'Failed to load files',
    'files.selected_count': '{count} selected',
    'files.select_all': 'Select All',
    'files.select_item': 'Select {name}',
    'files.actions_menu': 'Actions menu for {name}',
    'files.move_success': 'Items moved successfully',
    'files.move_conflict': 'A file or folder with the same name already exists in target (409).',
    'files.move_error': 'Failed to move items',
    'files.pagination_summary': 'Showing {start} - {end} of {total} items',
    'files.prev_page': 'Previous Page',
    'files.next_page': 'Next Page',
    'files.previous': 'Previous',
    'files.next': 'Next',

    // New Folder Modal
    'folder.create_title': 'Create New Folder',
    'folder.name_label': 'Folder Name',
    'folder.placeholder': 'e.g. Projects',
    'folder.empty_error': 'Folder name cannot be empty',
    'folder.invalid_chars': 'Folder name cannot contain / or \\',
    'folder.create_button': 'Create',
    'folder.creating': 'Creating...',

    // Upload Modal
    'upload.title': 'Upload File',
    'upload.dropzone_text': 'Drag and drop files here or click to browse',
    'upload.overwrite': 'Overwrite',
    'upload.conflict_warning': 'A file with the same name already exists in this location.',
    'upload.remove_from_list': 'Remove from list',
    'upload.no_files_selected': 'No files selected',
    'upload.files_selected': '{count} files selected',
    'upload.uploading': 'Uploading...',

    // Rename Modal
    'rename.title': 'Rename',
    'rename.name_label': 'New Name',
    'rename.empty_error': 'Name cannot be empty',
    'rename.invalid_chars': 'Name cannot contain / or \\',
    'rename.save_button': 'Save',

    // Move Modal
    'move.title': 'Move Item',
    'move.moving_item': 'Moving:',
    'move.target_dir_label': 'Target Folder Path',
    'move.example_path': 'Example: /documents or /',
    'move.same_folder_error': 'Target folder cannot be the same as the current folder',
    'move.submit_button': 'Move',

    // Delete Dialog
    'delete.title': 'Delete Item',
    'delete.single_confirm': 'Are you sure you want to delete {name} {type}?',
    'delete.multiple_confirm': 'Are you sure you want to delete {count} selected items?',
    'delete.folder': 'folder',
    'delete.file': 'file',
    'delete.submit_button': 'Delete',

    // Share & Guest
    'share.title': 'Create Share Link',
    'share.ready_title': 'Your Share Link is Ready',
    'share.slug_label': 'Custom Link Name (Slug)',
    'share.password_label': 'Access Password (Optional)',
    'share.burn_after_read': 'Delete this share after reading',
    'share.burn_description': 'Once accessed and read, the share link is permanently removed. The file itself is kept.',
    'share.submit': 'Create Share',
    'share.active': 'Active',
    'share.disabled': 'Disabled',
    'share.enable': 'Enable',
    'share.disable': 'Close',
    'share.guest_title': 'Depgo Share',
    'share.guest_burn_notice': 'This link is single-use: once you preview or download the file, the link will be deleted and cannot be reopened.',
    'share.unlock_button': 'Unlock Access',
    'share.password_placeholder': 'Password',
    'share.protected_title': 'Protected Share',
    'share.protected_desc': 'Please enter the access password to view this content.',
    'share.verifying': 'Verifying...',
    'share.no_active': 'No active share links yet',
    'share.no_active_desc': 'Select a file or folder from the Files tab and click "Share" to generate a link.',
    'share.link_copied': 'Link copied to clipboard',
    'share.copy_button': 'Copy Link',

    // Trashbin
    'trash.title': 'Trash Bin',
    'trash.subtitle': 'View, restore, or permanently purge deleted local files and folders.',
    'trash.empty': 'Trash bin is empty',
    'trash.empty_desc': 'Deleted files and folders will appear here.',
    'trash.original_path': 'Original Path & Name',
    'trash.deleted_at': 'Deleted At',
    'trash.restore': 'Restore',
    'trash.permanent_delete': 'Delete Permanently',
    'trash.collision_title': 'Restore Collision',
    'trash.collision_desc': 'An item with the same name already exists at the original location ({path}).',
    'trash.custom_path_label': 'Alternative Restore Path',
    'trash.restore_to_path': 'Restore to this Path',
    'trash.permanent_delete_title': 'Delete Permanently',
    'trash.permanent_delete_confirm': 'Are you sure you want to permanently delete {path}? This action cannot be undone.',

    // Editor & Versions
    'editor.content_label': 'Text editor content',
    'editor.modified_badge': 'Modified',
    'editor.save': 'Save',
    'editor.unsaved_warning': 'You have unsaved changes. Are you sure you want to close?',
    'versions.title': 'Version History',
    'versions.restore_confirm': 'Are you sure you want to revert to revision #{revision}? The current content will be preserved as a new version.',
    'versions.restore_button': 'Restore',
    'preview.preparing': 'Preparing preview...',

    // Errors
    'error.incorrect_credentials': 'Username or password is incorrect.',
    'error.incorrect_password': 'Password is incorrect.',
    'error.share_already_exists': 'This link name is already taken. Please choose another name.',
    'error.too_many_attempts': 'Too many unsuccessful attempts. Please try again later.',
    'error.share_not_found': 'This share could not be found. It may have been deleted, closed, or already consumed if it was single-use.',
    'error.share_expired': 'This share link has expired or reached its maximum usage limit.',
    'error.unexpected': 'An unexpected error occurred.',
    'error.conflict': 'An item with the same name already exists.',
  },
};

export function t(key: string, params?: Record<string, string | number>): string {
  const langDict = translations[currentLanguage] || translations.tr;
  let text = langDict[key] ?? translations.tr[key] ?? key;

  if (params) {
    Object.entries(params).forEach(([k, v]) => {
      text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v));
    });
  }

  return text;
}

export function useTranslation() {
  const [lang, setLangState] = useState<Language>(currentLanguage);

  useEffect(() => {
    return subscribeLanguage((newLang) => {
      setLangState(newLang);
    });
  }, []);

  return {
    t,
    lang,
    setLanguage,
  };
}
