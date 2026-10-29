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

    // Nav & Sidebar
    'nav.files': 'Dosyalar',
    'nav.shares': 'Paylaşımlar',
    'nav.trash': 'Çöp Kutusu',
    'nav.logout': 'Çıkış Yap',
    'nav.switch_lang': 'Dil Seçimi',
    'nav.view_list': 'Liste Görünümü',
    'nav.view_grid': 'Izgara Görünümü',
    'nav.storage_quota': 'Depolama',

    // Dosya İşlemleri & Araç Çubuğu
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
    'files.actions_menu': '{name} için işlemler menüsü',
    'files.move_success': 'Öğeler başarıyla taşındı',
    'files.move_conflict': 'Hedef klasörde aynı isimde dosya veya klasör zaten var (409).',
    'files.move_error': 'Taşıma işlemi başarısız oldu',
    'files.pagination_summary': 'Toplam {total} öğeden {start} - {end} arası gösteriliyor',
    'files.prev_page': 'Önceki Sayfa',
    'files.next_page': 'Sonraki Sayfa',

    // Errors (Backend hata mesajları eşlemesi)
    'error.incorrect_credentials': 'Kullanıcı adı veya parola hatalı.',
    'error.incorrect_password': 'Parola hatalı.',
    'error.share_already_exists': 'Bu bağlantı adı zaten kullanılıyor. Başka bir ad seçin.',
    'error.too_many_attempts': 'Çok fazla hatalı deneme yapıldı. Lütfen biraz sonra tekrar deneyin.',
    'error.share_not_found': 'Bu paylaşım bulunamadı. Silinmiş ya da kapatılmış olabilir; tek okumalık bir bağlantı ise zaten kullanılmış olabilir.',
    'error.share_expired': 'Bu paylaşımın süresi dolmuş veya kullanım hakkı bitmiş.',
    'error.unexpected': 'Beklenmeyen bir hata oluştu.',
    'error.conflict': 'Aynı ada sahip bir öğe zaten mevcut.',

    // Paylaşım & Misafir
    'share.title': 'Paylaşım Bağlantısı Oluştur',
    'share.ready_title': 'Paylaşım Bağlantınız Hazır',
    'share.slug_label': 'Özel Bağlantı Adı (Slug)',
    'share.password_label': 'Erişim Parolası (İsteğe bağlı)',
    'share.burn_after_read': 'Okunduktan sonra bu paylaşımı sil',
    'share.submit': 'Paylaşım Oluştur',
    'share.active': 'Aktif',
    'share.disabled': 'Devre Dışı',
    'share.enable': 'Etkinleştir',
    'share.disable': 'Devre Dışı Bırak',
    'share.guest_title': 'Depgo Paylaşımı',
    'share.guest_burn_notice': 'Bu bağlantı tek okumalıktır. Dosyayı görüntüledikten veya indirdikten sonra bağlantı otomatik olarak silinecektir.',
    'share.unlock_button': 'Erişimi Aç',
    'share.password_placeholder': 'Parola',
  },
  en: {
    // Genel
    'app.name': 'Depgo',
    'app.tagline': 'Lightweight and Fast Cloud Storage',
    'common.close': 'Close',
    'common.cancel': 'Cancel',
    'common.save': 'Save',
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

    // Nav & Sidebar
    'nav.files': 'Files',
    'nav.shares': 'Shares',
    'nav.trash': 'Trash',
    'nav.logout': 'Sign Out',
    'nav.switch_lang': 'Language',
    'nav.view_list': 'List View',
    'nav.view_grid': 'Grid View',
    'nav.storage_quota': 'Storage',

    // Dosya İşlemleri & Araç Çubuğu
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
    'files.actions_menu': 'Actions menu for {name}',
    'files.move_success': 'Items moved successfully',
    'files.move_conflict': 'A file or folder with the same name already exists in target (409).',
    'files.move_error': 'Failed to move items',
    'files.pagination_summary': 'Showing {start} - {end} of {total} items',
    'files.prev_page': 'Previous Page',
    'files.next_page': 'Next Page',

    // Errors
    'error.incorrect_credentials': 'Username or password is incorrect.',
    'error.incorrect_password': 'Password is incorrect.',
    'error.share_already_exists': 'This link name is already taken. Please choose another name.',
    'error.too_many_attempts': 'Too many unsuccessful attempts. Please try again later.',
    'error.share_not_found': 'This share could not be found. It may have been deleted, closed, or already consumed if it was single-use.',
    'error.share_expired': 'This share link has expired or reached its maximum usage limit.',
    'error.unexpected': 'An unexpected error occurred.',
    'error.conflict': 'An item with the same name already exists.',

    // Paylaşım & Misafir
    'share.title': 'Create Share Link',
    'share.ready_title': 'Your Share Link is Ready',
    'share.slug_label': 'Custom Link Name (Slug)',
    'share.password_label': 'Access Password (Optional)',
    'share.burn_after_read': 'Delete this share after reading',
    'share.submit': 'Create Share',
    'share.active': 'Active',
    'share.disabled': 'Disabled',
    'share.enable': 'Enable',
    'share.disable': 'Disable',
    'share.guest_title': 'Depgo Share',
    'share.guest_burn_notice': 'This link is single-use. After viewing or downloading the file, the link will be automatically deleted.',
    'share.unlock_button': 'Unlock Access',
    'share.password_placeholder': 'Password',
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
