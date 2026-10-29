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
if (typeof document !== 'undefined') {
  document.documentElement.lang = currentLanguage;
}
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
  if (typeof document !== 'undefined') {
    document.documentElement.lang = lang;
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
    'common.update': 'Güncelle',
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
    'nav.breadcrumb_aria': 'Klasör Gezintisi',
    'nav.main_menu': 'Ana Menü',
    'nav.lang_tr': 'Türkçe',
    'nav.lang_en': 'English',
    'nav.root_dir': 'Kök Dizin',

    // Giriş (Login)
    'login.title': 'Depgo Girişi',
    'login.subtitle': 'Devam etmek için yönetici kimliğinizi doğrulayın.',
    'login.username_label': 'Kullanıcı Adı',
    'login.password_label': 'Şifre',
    'login.submit_button': 'Giriş Yap',
    'login.submitting': 'Giriş yapılıyor...',
    'login.required_fields': 'Lütfen kullanıcı adı ve şifrenizi girin',
    'login.failed': 'Giriş yapılamadı. Bilgilerinizi kontrol edin.',

    // Dosya Gezgini & Araç Çubuğu
    'files.new_folder': 'Yeni Klasör',
    'files.upload': 'Dosya Yükle',
    'files.upload_start': 'Yüklemeyi Başlat',
    'files.filter_placeholder': 'Dosya veya klasör ara...',
    'files.filter_aria': 'Dosyaları filtrele',
    'files.filter_clear': 'Filtreyi temizle',
    'files.name': 'Ad',
    'files.size': 'Boyut',
    'files.modified': 'Değiştirilme',
    'files.folder': 'Klasör',
    'files.empty_directory': 'Bu klasörde görüntülenecek öğe bulunamadı.',
    'files.loading_files': 'Dosyalar yükleniyor...',
    'files.load_failed': 'Dosyalar yüklenemedi',
    'files.selected_count': '{count} öğe seçildi',
    'files.item_count': '{count} öğe',
    'files.select_all': 'Tümünü Seç',
    'files.select_item': '{name} seç',
    'files.actions_menu': '{name} işlemler menüsü',
    'files.move_success': 'Öğeler başarıyla taşındı',
    'files.move_conflict': 'Hedef klasörde aynı isimde dosya veya klasör zaten var (409).',
    'files.move_error': 'Taşıma işlemi başarısız oldu',
    'files.download_url_failed': 'İndirme bağlantısı alınamadı',
    'files.selection_bar_aria': 'Çoklu seçim işlemleri',
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
    'folder.create_failed': 'Klasör oluşturulamadı',

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
    'rename.failed': 'Yeniden adlandırılamadı',

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
    'delete.failed': 'Silme işlemi başarısız',

    // Paylaşım Yönetimi & Modalı
    'share.title': 'Paylaşım Bağlantısı Oluştur',
    'share.edit_title': 'Paylaşım Bağlantısını Düzenle',
    'share.ready_title': 'Paylaşım Bağlantınız Hazır',
    'share.header_title': 'Paylaşım Bağlantıları',
    'share.header_desc': 'Dış kullanıcılar için oluşturulan dosya ve klasör bağlantılarını yönetin.',
    'share.slug_label': 'Özel Bağlantı Adı (Slug)',
    'share.slug_invalid': 'Özel bağlantı adı (slug) 1-32 karakter uzunluğunda olmalı ve yalnızca harf, rakam, alt çizgi veya tire içermelidir.',
    'share.name_label': 'Paylaşım Başlığı',
    'share.title_placeholder': 'Başlık girin',
    'share.password_label': 'Erişim Parolası (İsteğe bağlı)',
    'share.pwd_placeholder': 'Parola belirleyin',
    'share.pwd_keep_placeholder': 'Yeni parola girin (mevcut parolayı korumak için boş bırakın)',
    'share.remove_password': 'Parolayı kaldır (bağlantıyı bilen herkes açabilir)',
    'share.expiration_label': 'Son Kullanma Süresi',
    'share.exp_unlimited': 'Süresiz',
    'share.exp_1h': '1 Saat',
    'share.exp_24h': '24 Saat (1 Gün)',
    'share.exp_3d': '3 Gün',
    'share.exp_7d': '7 Gün (1 Hafta)',
    'share.exp_30d': '30 Gün (1 Ay)',
    'share.allow_preview': 'Tarayıcı içi önizlemeye izin ver',
    'share.allow_download': 'İndirmeye izin ver',
    'share.burn_after_read': 'Okunduktan sonra bu paylaşımı sil',
    'share.burn_description': 'Bağlantı bir kez açılıp içeriği alınınca paylaşım silinir. Dosyanın kendisi silinmez.',
    'share.burn_notice_short': 'Bu bağlantı tek okumalık: bir kez kullanıldıktan sonra otomatik olarak silinecektir.',
    'share.burn_badge': 'Tek okumalık',
    'share.submit': 'Paylaşım Oluştur',
    'share.active': 'Aktif',
    'share.disabled': 'Devre Dışı',
    'share.enable': 'Etkinleştir',
    'share.disable': 'Kapat',
    'share.enable_title': 'Bağlantıyı Yeniden Etkinleştir',
    'share.disable_title': 'Bağlantıyı Devre Dışı Bırak',
    'share.disable_confirm': '"{name}" bağlantısı kapatılsın mı? Bağlantı çalışmayı bırakır ama kaydı ve adresi kalır; istediğin zaman yeniden etkinleştirebilirsin.',
    'share.disable_failed': 'Paylaşım kapatılamadı',
    'share.enable_spent_confirm': 'Bu bağlantının erişim hakkı dolmuştu. Yeniden etkinleştirince hak sıfırlanır ve bağlantı tekrar okunabilir. Devam edilsin mi?',
    'share.enable_failed': 'Paylaşım etkinleştirilemedi',
    'share.delete_confirm': 'Bu paylaşım bağlantısını tamamen silmek istediğinize emin misiniz?',
    'share.delete_failed': 'Paylaşım silinemedi',
    'share.failed': 'Paylaşım işlemi başarısız',
    'share.load_failed': 'Paylaşımlar yüklenemedi',
    'share.loading_shares': 'Paylaşımlar yükleniyor...',
    'share.th_link_name': 'Bağlantı & Ad',
    'share.th_root_path': 'Kök Yol',
    'share.th_security': 'Güvenlik',
    'share.th_expiration': 'Son Kullanma',
    'share.th_views': 'Görüntüleme',
    'share.th_status': 'Durum',
    'share.th_actions': 'İşlemler',
    'share.badge_password': 'Parolalı',
    'share.badge_public': 'Genel',
    'share.view_count_badge': '{count} kez',
    'share.pagination_summary': 'Toplam {total} paylaşımdan {start} - {end} arası gösteriliyor',
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

    // Misafir (Guest) Görünümü
    'guest.info_failed': 'Paylaşım bilgisi alınamadı',
    'guest.dir_failed': 'Klasör içeriği yüklenemedi',
    'guest.enter_password': 'Lütfen parolayı girin',
    'guest.access_failed_title': 'Paylaşıma Erişilemedi',
    'guest.shared_label': 'Paylaşılan: {name}',
    'guest.loading_content': 'İçerik yükleniyor...',

    // Çöp Kutusu (Trash)
    'trash.title': 'Çöp Kutusu',
    'trash.subtitle': 'Silinmiş yerel dosya ve klasörleri görüntüleyin, geri yükleyin veya kalıcı olarak temizleyin.',
    'trash.load_failed': 'Çöp kutusu listesi alınamadı',
    'trash.loading': 'Çöp kutusu yükleniyor...',
    'trash.empty': 'Çöp kutusu boş',
    'trash.empty_desc': 'Silinen öğeler burada saklanır.',
    'trash.original_path': 'Orijinal Yol & Ad',
    'trash.deleted_at': 'Silinme Tarihi',
    'trash.restore': 'Geri Yükle',
    'trash.restore_failed': 'Geri yükleme başarısız',
    'trash.permanent_delete': 'Kalıcı Olarak Sil',
    'trash.delete_failed': 'Kalıcı silme işlemi başarısız',
    'trash.collision_title': 'Geri Yükleme Çakışması',
    'trash.collision_desc': 'Orijinal konumda ({path}) aynı isimde bir öğe zaten mevcut.',
    'trash.collision_error': 'Orijinal konumda aynı isimde bir dosya veya klasör mevcut. Lütfen alternatif bir geri yükleme yolu girin.',
    'trash.custom_path_label': 'Alternatif Geri Yükleme Yolu',
    'trash.restore_to_path': 'Bu Konuma Geri Yükle',
    'trash.permanent_delete_title': 'Kalıcı Olarak Sil',
    'trash.permanent_delete_confirm': '{path} öğesini kalıcı olarak silmek istediğinize emin misiniz? Bu işlem geri alınamaz.',

    // Metin Editörü & Sürüm Geçmişi
    'editor.content_label': 'Metin düzenleyici içeriği',
    'editor.modified_badge': 'Değiştirildi',
    'editor.save': 'Kaydet',
    'editor.unsaved_warning': 'Kaydedilmemiş değişiklikleriniz var. Kapatmak istediğinize emin misiniz?',
    'editor.unsaved_reload_warning': 'Mevcut düzenlemeleriniz kaybolacak. Devam edilsin mi?',
    'editor.load_failed': 'Metin içeriği yüklenemedi',
    'editor.save_failed': 'Kayıt işlemi başarısız',
    'editor.conflict_desc': 'Çakışma tespit edildi: Dosya siz düzenlerken başka bir işlem veya kullanıcı tarafından değiştirildi. Kaydetmek mevcut sürümün üzerine yazılmasına neden olabilir. Lütfen güncel içeriği tekrar yükleyin.',
    'editor.reload': 'Yeniden Yükle',
    'editor.loading': 'Metin yükleniyor...',
    'editor.empty_placeholder': 'Dosya içeriği boş...',
    'editor.line_count': 'Satır',
    'editor.unsupported_ext': 'Bu dosya uzantısı metin düzenleyici tarafından desteklenmiyor.',
    'editor.size_limit_exceeded': 'Dosya boyutu 2 MB metin düzenleme sınırını aşıyor ({formattedSize}).',
    'editor.size_limit_exceeded_editor': 'Metin boyutu 5 MB sınırını aşıyor ({formattedSize}).',
    'versions.title': 'Sürüm Geçmişi',
    'versions.load_failed': 'Sürüm geçmişi alınamadı',
    'versions.restore_failed': 'Sürüm geri yüklenemedi',
    'versions.restore_confirm': 'Revizyon #{revision} sürümüne dönmek istediğinize emin misiniz? Mevcut içerik geçmişe kaydedilerek korunacaktır.',
    'versions.restore_button': 'Geri Dön',
    'versions.conflict_desc': 'Sürüm çakışması (409/412): Dosya başka bir işlem tarafından değiştirilmiş. Lütfen güncel sürüm listesini yenileyip tekrar deneyin.',
    'versions.file_label': 'Dosya',
    'versions.active_rev': 'Aktif Rev #{revision}',
    'versions.loading': 'Sürümler yükleniyor...',
    'versions.empty': 'Bu dosya için kaydedilmiş önceki bir sürüm bulunamadı.',
    'versions.revision': 'Revizyon',
    'versions.saved_date': 'Kayıt Tarihi',
    'versions.current': 'Mevcut',
    'versions.total_versions': 'Toplam {count} sürüm',

    // Önizleme (Preview)
    'preview.preparing': 'Önizleme hazırlanıyor...',
    'preview.preparing_securely': 'Önizleme güvenli şekilde hazırlanıyor...',
    'preview.svg_html_warning': 'Güvenlik nedeniyle SVG ve HTML dosyaları önizlenmez, doğrudan indirilir.',
    'preview.svg_html_browser': 'Güvenlik nedeniyle SVG ve HTML dosyaları tarayıcıda önizlenmez. Lütfen dosyayı indirin.',
    'preview.unsupported_type': 'Bu dosya türü için tarayıcı içi önizleme desteklenmiyor.',
    'preview.size_limit_exceeded': 'Dosya boyutu 20 MB sınırını aşıyor ({formattedSize}). Lütfen dosyayı indirin.',
    'preview.url_not_found': 'Önizleme adresi bulunamadı.',
    'preview.load_failed': 'Önizleme yüklenemedi',
    'preview.download_file': 'Dosyayı İndir',
    'preview.pdf_unsupported': 'Tarayıcınız yerleşik PDF görüntüleyiciyi desteklemiyor.',
    'preview.download_pdf': "PDF'i İndir",
    'preview.url_fetch_failed': 'Önizleme bağlantısı alınamadı',

    // Güvenlik (Security)
    'security.dangerous_mime': 'Güvenlik nedeniyle HTML, SVG veya komut dosyası içerikleri önizlenemez.',
    'security.html_svg_detected': 'Dosya içeriği HTML/SVG olarak tespit edildi; güvenlik nedeniyle önizlenemez.',
    'security.invalid_pdf_signature': 'Geçersiz PDF dosyası: Dosya başında geçerli bir PDF imzası (%PDF-) bulunamadı.',
    'security.invalid_image_signature': 'Geçersiz resim formatı: Desteklenen raster resim imzası (PNG, JPEG, GIF, WebP) bulunamadı.',
    'security.unsupported_format': 'Desteklenmeyen dosya formatı.',

    // Hata Mesajları
    'error.incorrect_credentials': 'Kullanıcı adı veya parola hatalı.',
    'error.incorrect_password': 'Parola hatalı.',
    'error.share_already_exists': 'Bu bağlantı adı zaten kullanılıyor. Başka bir ad seçin.',
    'error.too_many_attempts': 'Çok fazla hatalı deneme yapıldı. Lütfen biraz sonra tekrar deneyin.',
    'error.share_not_found': 'Bu paylaşım bulunamadı. Silinmiş ya da kapatılmış olabilir; tek okumalık bir bağlantı ise zaten kullanılmış olabilir.',
    'error.share_expired': 'Bu paylaşımın süresi dolmuş veya kullanım hakkı bitmiş.',
    'error.unexpected': 'Beklenmeyen bir hata oluştu.',
    'error.conflict': 'Aynı ada sahip bir öğe zaten mevcut.',
    'error.session_expired': 'Oturum süresi doldu veya yetkisiz erişim',
    'error.server_error': 'Sunucu hatası: {status}',
    'error.invalid_json': 'Sunucudan geçersiz JSON yanıt alındı',
    'error.session_invalid': 'Oturum geçersiz',
    'error.operation_failed': 'İşlem başarısız',
    'error.session_auth_error': 'Oturum yetkilendirme hatası',
    'error.upload_failed': 'Yükleme başarısız',
    'error.upload_error_status': 'Yükleme hatası: HTTP {status}',
    'error.network_upload_failed': 'Ağ bağlantısı hatası nedeniyle dosya yüklenemedi',
    'error.upload_cancelled': 'Yükleme iptal edildi',
    'error.preview_security_origin': 'Güvenlik uyarısı: Yalnızca depgo aynı kökenli (/d/, /p/, /sd/, /sp/) dahili indirme yolları önizlenebilir.',
    'error.preview_cancelled': 'Önizleme isteği iptal edildi',
    'error.preview_rejected_network': 'Önizleme isteği yönlendirme veya ağ hatası nedeniyle reddedildi',
    'error.unauthorized': 'Yetkisiz erişim',
    'error.file_fetch_failed': 'Dosya alınamadı: HTTP {status}',
    'error.preview_redirect_forbidden': 'Güvenlik gerekçesiyle yönlendirilmiş önizleme adresleri kabul edilmez.',
    'error.response_body_unread': 'Yanıt gövdesi okunamadı',
    'error.preview_size_exceeded': 'Dosya boyutu {limit} MB sınırını aşıyor.',
    'error.request_cancelled': 'İstek iptal edildi',
    'error.stream_read_error': 'Dosya akışı okunurken hata oluştu',
    'error.empty_preview': 'Boş dosya içeriği önizlenemez',
    'error.invalid_preview_format': 'Dosya içeriği geçerli bir önizleme formatına uymuyor.',
  },
  en: {
    // General
    'app.name': 'Depgo',
    'app.tagline': 'Lightweight and Fast Cloud Storage',
    'common.close': 'Close',
    'common.cancel': 'Cancel',
    'common.save': 'Save',
    'common.saving': 'Saving...',
    'common.update': 'Update',
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
    'nav.breadcrumb_aria': 'Breadcrumbs navigation',
    'nav.main_menu': 'Main Menu',
    'nav.lang_tr': 'Turkish',
    'nav.lang_en': 'English',
    'nav.root_dir': 'Root Directory',

    // Login
    'login.title': 'Depgo Sign In',
    'login.subtitle': 'Authenticate your administrator identity to continue.',
    'login.username_label': 'Username',
    'login.password_label': 'Password',
    'login.submit_button': 'Sign In',
    'login.submitting': 'Signing in...',
    'login.required_fields': 'Please enter username and password',
    'login.failed': 'Sign in failed. Please check your credentials.',

    // Files Explorer & Toolbar
    'files.new_folder': 'New Folder',
    'files.upload': 'Upload File',
    'files.upload_start': 'Start Upload',
    'files.filter_placeholder': 'Search files or folders...',
    'files.filter_aria': 'Filter files',
    'files.filter_clear': 'Clear filter',
    'files.name': 'Name',
    'files.size': 'Size',
    'files.modified': 'Modified',
    'files.folder': 'Folder',
    'files.empty_directory': 'No items to display in this folder.',
    'files.loading_files': 'Loading files...',
    'files.load_failed': 'Failed to load files',
    'files.selected_count': '{count} selected',
    'files.item_count': '{count} items',
    'files.select_all': 'Select All',
    'files.select_item': 'Select {name}',
    'files.actions_menu': 'Actions menu for {name}',
    'files.move_success': 'Items moved successfully',
    'files.move_conflict': 'A file or folder with the same name already exists in target (409).',
    'files.move_error': 'Failed to move items',
    'files.download_url_failed': 'Could not obtain download link',
    'files.selection_bar_aria': 'Multi-selection actions',
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
    'folder.create_failed': 'Failed to create folder',

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
    'rename.failed': 'Failed to rename',

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
    'delete.failed': 'Delete operation failed',

    // Share & Guest
    'share.title': 'Create Share Link',
    'share.edit_title': 'Edit Share Link',
    'share.ready_title': 'Your Share Link is Ready',
    'share.header_title': 'Share Links',
    'share.header_desc': 'Manage file and folder links generated for external users.',
    'share.slug_label': 'Custom Link Name (Slug)',
    'share.slug_invalid': 'Custom slug must be 1-32 characters long and contain only letters, numbers, underscores, or dashes.',
    'share.name_label': 'Share Title',
    'share.title_placeholder': 'Enter title',
    'share.password_label': 'Access Password (Optional)',
    'share.pwd_placeholder': 'Set a password',
    'share.pwd_keep_placeholder': 'Enter new password (leave blank to keep current)',
    'share.remove_password': 'Remove password (anyone with the link can open)',
    'share.expiration_label': 'Expiration Time',
    'share.exp_unlimited': 'Unlimited',
    'share.exp_1h': '1 Hour',
    'share.exp_24h': '24 Hours (1 Day)',
    'share.exp_3d': '3 Days',
    'share.exp_7d': '7 Days (1 Week)',
    'share.exp_30d': '30 Days (1 Month)',
    'share.allow_preview': 'Allow in-browser preview',
    'share.allow_download': 'Allow download',
    'share.burn_after_read': 'Delete this share after reading',
    'share.burn_description': 'Once accessed and read, the share link is permanently removed. The file itself is kept.',
    'share.burn_notice_short': 'This link is single-use: it will be automatically deleted after first use.',
    'share.burn_badge': 'Single-use',
    'share.submit': 'Create Share',
    'share.active': 'Active',
    'share.disabled': 'Disabled',
    'share.enable': 'Enable',
    'share.disable': 'Disable',
    'share.enable_title': 'Re-enable link',
    'share.disable_title': 'Disable link',
    'share.disable_confirm': 'Disable link "{name}"? The link will stop working but stays saved; you can re-enable it anytime.',
    'share.disable_failed': 'Failed to disable share',
    'share.enable_spent_confirm': 'This link had reached its usage limit. Re-enabling will reset the limit so it can be accessed again. Continue?',
    'share.enable_failed': 'Failed to enable share',
    'share.delete_confirm': 'Are you sure you want to permanently delete this share link?',
    'share.delete_failed': 'Failed to delete share',
    'share.failed': 'Share operation failed',
    'share.load_failed': 'Failed to load shares',
    'share.loading_shares': 'Loading shares...',
    'share.th_link_name': 'Link & Name',
    'share.th_root_path': 'Root Path',
    'share.th_security': 'Security',
    'share.th_expiration': 'Expiration',
    'share.th_views': 'Views',
    'share.th_status': 'Status',
    'share.th_actions': 'Actions',
    'share.badge_password': 'Password',
    'share.badge_public': 'Public',
    'share.view_count_badge': '{count} times',
    'share.pagination_summary': 'Showing {start} - {end} of {total} shares',
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

    // Guest View
    'guest.info_failed': 'Failed to get share info',
    'guest.dir_failed': 'Failed to load folder content',
    'guest.enter_password': 'Please enter password',
    'guest.access_failed_title': 'Share Inaccessible',
    'guest.shared_label': 'Shared: {name}',
    'guest.loading_content': 'Loading content...',

    // Trashbin
    'trash.title': 'Trash Bin',
    'trash.subtitle': 'View, restore, or permanently purge deleted local files and folders.',
    'trash.load_failed': 'Failed to load trash list',
    'trash.loading': 'Loading trash...',
    'trash.empty': 'Trash bin is empty',
    'trash.empty_desc': 'Deleted files and folders will appear here.',
    'trash.original_path': 'Original Path & Name',
    'trash.deleted_at': 'Deleted At',
    'trash.restore': 'Restore',
    'trash.restore_failed': 'Failed to restore',
    'trash.permanent_delete': 'Delete Permanently',
    'trash.delete_failed': 'Failed to permanently delete',
    'trash.collision_title': 'Restore Collision',
    'trash.collision_desc': 'An item with the same name already exists at the original location ({path}).',
    'trash.collision_error': 'An item with the same name exists at original path. Please enter an alternative restore path.',
    'trash.custom_path_label': 'Alternative Restore Path',
    'trash.restore_to_path': 'Restore to this Path',
    'trash.permanent_delete_title': 'Delete Permanently',
    'trash.permanent_delete_confirm': 'Are you sure you want to permanently delete {path}? This action cannot be undone.',

    // Editor & Versions
    'editor.content_label': 'Text editor content',
    'editor.modified_badge': 'Modified',
    'editor.save': 'Save',
    'editor.unsaved_warning': 'You have unsaved changes. Are you sure you want to close?',
    'editor.unsaved_reload_warning': 'Your current unsaved changes will be lost. Continue?',
    'editor.load_failed': 'Failed to load text content',
    'editor.save_failed': 'Failed to save',
    'editor.conflict_desc': 'Conflict detected: File was modified by another process while you were editing. Saving may overwrite recent changes. Please reload the current content.',
    'editor.reload': 'Reload',
    'editor.loading': 'Loading text...',
    'editor.empty_placeholder': 'File content is empty...',
    'editor.line_count': 'Lines',
    'editor.unsupported_ext': 'This file extension is not supported by the text editor.',
    'editor.size_limit_exceeded': 'File size exceeds the 2 MB text editing limit ({formattedSize}).',
    'editor.size_limit_exceeded_editor': 'Text size exceeds 5 MB limit ({formattedSize}).',
    'versions.title': 'Version History',
    'versions.load_failed': 'Failed to load version history',
    'versions.restore_failed': 'Failed to restore version',
    'versions.restore_confirm': 'Are you sure you want to revert to revision #{revision}? The current content will be preserved as a new version.',
    'versions.restore_button': 'Restore',
    'versions.conflict_desc': 'Version conflict (409/412): File was modified by another operation. Please refresh the version list and try again.',
    'versions.file_label': 'File',
    'versions.active_rev': 'Active Rev #{revision}',
    'versions.loading': 'Loading versions...',
    'versions.empty': 'No previous versions found for this file.',
    'versions.revision': 'Revision',
    'versions.saved_date': 'Saved Date',
    'versions.current': 'Current',
    'versions.total_versions': 'Total {count} versions',

    // Preview
    'preview.preparing': 'Preparing preview...',
    'preview.preparing_securely': 'Preparing preview securely...',
    'preview.svg_html_warning': 'For security reasons, SVG and HTML files cannot be previewed; please download them directly.',
    'preview.svg_html_browser': 'For security reasons, SVG and HTML files are not previewed in the browser. Please download the file.',
    'preview.unsupported_type': 'In-browser preview is not supported for this file type.',
    'preview.size_limit_exceeded': 'File size exceeds the 20 MB limit ({formattedSize}). Please download the file.',
    'preview.url_not_found': 'Preview URL not found.',
    'preview.load_failed': 'Failed to load preview',
    'preview.download_file': 'Download File',
    'preview.pdf_unsupported': 'Your browser does not support built-in PDF viewer.',
    'preview.download_pdf': 'Download PDF',
    'preview.url_fetch_failed': 'Could not obtain preview link',

    // Security
    'security.dangerous_mime': 'For security reasons, HTML, SVG or script contents cannot be previewed.',
    'security.html_svg_detected': 'File content detected as HTML/SVG; cannot be previewed for security reasons.',
    'security.invalid_pdf_signature': 'Invalid PDF file: Valid PDF signature (%PDF-) not found at beginning of file.',
    'security.invalid_image_signature': 'Invalid image format: Supported raster image signature (PNG, JPEG, GIF, WebP) not found.',
    'security.unsupported_format': 'Unsupported file format.',

    // Errors
    'error.incorrect_credentials': 'Username or password is incorrect.',
    'error.incorrect_password': 'Password is incorrect.',
    'error.share_already_exists': 'This link name is already taken. Please choose another name.',
    'error.too_many_attempts': 'Too many unsuccessful attempts. Please try again later.',
    'error.share_not_found': 'This share could not be found. It may have been deleted, closed, or already consumed if it was single-use.',
    'error.share_expired': 'This share link has expired or reached its maximum usage limit.',
    'error.unexpected': 'An unexpected error occurred.',
    'error.conflict': 'An item with the same name already exists.',
    'error.session_expired': 'Session expired or unauthorized access',
    'error.server_error': 'Server error: {status}',
    'error.invalid_json': 'Invalid JSON response received from server',
    'error.session_invalid': 'Session is invalid',
    'error.operation_failed': 'Operation failed',
    'error.session_auth_error': 'Session authentication error',
    'error.upload_failed': 'Upload failed',
    'error.upload_error_status': 'Upload error: HTTP {status}',
    'error.network_upload_failed': 'File could not be uploaded due to a network connection error',
    'error.upload_cancelled': 'Upload cancelled',
    'error.preview_security_origin': 'Security warning: Only depgo same-origin (/d/, /p/, /sd/, /sp/) internal download paths can be previewed.',
    'error.preview_cancelled': 'Preview request cancelled',
    'error.preview_rejected_network': 'Preview request rejected due to redirect or network error',
    'error.unauthorized': 'Unauthorized access',
    'error.file_fetch_failed': 'Could not retrieve file: HTTP {status}',
    'error.preview_redirect_forbidden': 'Redirected preview URLs are not permitted for security reasons.',
    'error.response_body_unread': 'Could not read response body',
    'error.preview_size_exceeded': 'File size exceeds {limit} MB limit.',
    'error.request_cancelled': 'Request cancelled',
    'error.stream_read_error': 'An error occurred while reading the file stream',
    'error.empty_preview': 'Empty file content cannot be previewed',
    'error.invalid_preview_format': 'File content does not match a valid preview format.',
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
