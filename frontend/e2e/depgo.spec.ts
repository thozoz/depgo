import { test, expect, Page, APIRequestContext } from '@playwright/test';
import path from 'node:path';
import fs from 'node:fs';
import { VISIBLE_PNG_BUFFER, VISIBLE_PDF_BUFFER, SAMPLE_TEXT_BUFFER } from './fixtures';

const screenshots = path.resolve('test-results/screenshots');
const run = Date.now().toString(36);
const folder = `_e2e_${run}`;
const root = `/${folder}`;
const shareIds = new Set<string>();
let token = '';
let api: APIRequestContext;

async function call(route: string, data?: object, method = 'POST', expected = 200) {
  const response = await api.fetch(route, {
    method,
    headers: { Authorization: token },
    ...(data ? { data } : {}),
  });
  const envelope = await response.json();
  expect(envelope.code, `${route}: ${envelope.message}`).toBe(expected);
  return envelope.data;
}

async function upload(name: string, content: Buffer) {
  const response = await api.put('/api/fs/put', {
    headers: {
      Authorization: token,
      'File-Path': encodeURIComponent(`${root}/${name}`),
      'As-Task': 'false',
      Overwrite: 'false',
      'Content-Type': 'application/octet-stream',
    },
    data: content,
  });
  const result = await response.json();
  expect(result.code, result.message).toBe(200);
}

async function login(page: Page) {
  await page.goto('/');
  await page.getByLabel('Kullanıcı Adı').fill(process.env.DEPGO_E2E_USERNAME || 'admin');
  await page.getByLabel('Şifre').fill(process.env.DEPGO_E2E_PASSWORD!);
  await page.getByRole('button', { name: 'Giriş Yap', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Giriş Yap', exact: true })).toBeHidden();
}

async function enterFolder(page: Page) {
  await page.getByRole('button', { name: folder, exact: true }).click();
  await expect(page.getByRole('button', { name: 'document.txt', exact: true })).toBeVisible();
}

function row(page: Page, name: string) {
  return page.locator('tbody tr').filter({ has: page.getByRole('button', { name, exact: true }) });
}

async function screenshot(page: Page, name: string) {
  await page.screenshot({ path: path.join(screenshots, `${name}.png`), fullPage: true });
}

const text = () => call(`/api/fs/text?path=${encodeURIComponent(`${root}/document.txt`)}`, undefined, 'GET');
const versions = () => call(`/api/fs/versions?path=${encodeURIComponent(`${root}/document.txt`)}`, undefined, 'GET');

test.describe.serial('Depgo real browser lifecycle', () => {
  test.beforeAll(async ({ playwright }) => {
    if (!process.env.DEPGO_E2E_PASSWORD) throw new Error('DEPGO_E2E_PASSWORD is required');
    fs.mkdirSync(screenshots, { recursive: true });
    api = await playwright.request.newContext({ baseURL: process.env.DEPGO_BASE_URL || 'http://localhost:15244' });
    const response = await api.post('/api/auth/login', {
      data: { username: process.env.DEPGO_E2E_USERNAME || 'admin', password: process.env.DEPGO_E2E_PASSWORD },
    });
    const result = await response.json();
    expect(result.code, result.message).toBe(200);
    token = result.data.token;
    await call('/api/fs/mkdir', { path: root });
    await upload('document.txt', SAMPLE_TEXT_BUFFER);
    await upload('collision.txt', Buffer.from('Original trash content'));
  });

  test.beforeEach(async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('console', message => {
      if (message.type() === 'error' && !/Failed to load resource:.*\b(403|404|409|410|412)\b/.test(message.text())) {
        errors.push(message.text());
      }
    });
    page.on('requestfailed', request => {
      if (!/ERR_ABORTED/.test(request.failure()?.errorText || '')) errors.push(`${request.method()} ${new URL(request.url()).pathname}: ${request.failure()?.errorText}`);
    });
    page.on('response', response => {
      if (response.status() >= 500) errors.push(`HTTP ${response.status()} ${new URL(response.url()).pathname}`);
    });
    test.info().annotations.push({ type: 'runtime-errors', description: JSON.stringify(errors) });
    (page as Page & { runtimeErrors?: string[] }).runtimeErrors = errors;
    await login(page);
    await enterFolder(page);
  });

  test.afterEach(async ({ page }) => {
    expect((page as Page & { runtimeErrors?: string[] }).runtimeErrors, 'browser console/network errors').toEqual([]);
  });

  test.afterAll(async () => {
    if (!api || !token) return;
    try {
      for (const share_id of shareIds) {
        await api.post('/api/share/delete', { headers: { Authorization: token }, data: { share_id } });
      }
      await api.post('/api/fs/remove', {
        headers: { Authorization: token }, data: { dir: '/', names: [folder] },
      });
      const response = await api.get('/api/fs/trash?path=/&per_page=100', { headers: { Authorization: token } });
      const result = await response.json();
      for (const item of result.data?.content || []) {
        if (item.original_path === root || item.original_path.startsWith(`${root}/`)) {
          await api.post('/api/fs/trash/delete', {
            headers: { Authorization: token }, data: { path: '/', trash_id: item.trash_id, confirm: true },
          });
        }
      }
    } finally {
      await api.dispose();
    }
  });

  test('desktop upload, visible image and PDF preview', async ({ page }) => {
    await page.getByRole('button', { name: 'Dosya Yükle', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Dosya Yükle', exact: true });
    await dialog.locator('input[type=file]').setInputFiles([
      { name: 'picture.png', mimeType: 'image/png', buffer: VISIBLE_PNG_BUFFER },
      { name: 'reader.pdf', mimeType: 'application/pdf', buffer: VISIBLE_PDF_BUFFER },
    ]);
    await dialog.getByRole('button', { name: 'Yüklemeyi Başlat' }).click();
    const done = dialog.getByText('Kapat', { exact: true });
    await expect(done).toBeEnabled();
    await done.click();
    await expect(row(page, 'reader.pdf')).toBeVisible();
    await screenshot(page, 'desktop-files');

    await row(page, 'picture.png').getByTitle('Önizle', { exact: true }).click();
    const image = page.getByRole('dialog', { name: 'picture.png' });
    await expect(image.locator('img')).toBeVisible();
    await expect.poll(() => image.locator('img').evaluate((element: HTMLImageElement) => element.naturalWidth)).toBe(400);
    await screenshot(page, 'image-preview');
    await image.getByLabel('Kapat', { exact: true }).click();

    await row(page, 'reader.pdf').getByTitle('Önizle', { exact: true }).click();
    const pdf = page.getByRole('dialog', { name: 'reader.pdf' });
    await expect(pdf.locator('object')).toBeVisible();
    await expect(pdf.getByText('Önizleme hazırlanıyor...')).toBeHidden();
    const blob = await pdf.locator('object').getAttribute('data');
    expect(blob).toMatch(/^blob:/);
    expect(await page.evaluate(async url => (await (await fetch(url!)).text()).includes('Depgo PDF preview test'), blob)).toBe(true);
    await screenshot(page, 'pdf-preview');
    // Native PDF plugins are browser-dependent; the screenshot is reviewed separately.
    await pdf.getByLabel('Kapat', { exact: true }).click();
  });

  test('mobile menu exposes all actions without overflow', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await row(page, 'document.txt').getByLabel(/işlemler menüsü/i).click();
    const menu = page.getByRole('menu');
    await expect(menu).toBeVisible();
    for (const name of ['Paylaş', 'Yeniden Adlandır', 'Sil']) {
      await expect(menu.getByRole('menuitem', { name, exact: true })).toBeVisible();
    }
    const bounds = await menu.boundingBox();
    expect(bounds!.x).toBeGreaterThanOrEqual(0);
    expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    await screenshot(page, 'mobile-files');
    await menu.getByRole('menuitem', { name: 'Yeniden Adlandır', exact: true }).click();
    await expect(page.getByRole('dialog', { name: 'Yeniden Adlandır', exact: true })).toBeVisible();
    await page.getByRole('dialog').getByLabel('Kapat', { exact: true }).click();
  });

  test('text edit, revision conflict and restoring a historical version', async ({ page }) => {
    const before = await text();
    const changed = `${before.content}\nSaved through actual browser.`;
    await row(page, 'document.txt').getByTitle('Metni Düzenle', { exact: true }).click();
    const editor = page.getByRole('dialog', { name: 'document.txt' });
    const textarea = editor.getByLabel('Metin düzenleyici içeriği');
    await expect(textarea).toHaveValue(before.content);
    await textarea.fill(changed);
    await screenshot(page, 'text-editor');
    await editor.getByRole('button', { name: 'Kaydet', exact: true }).click();
    await expect(editor.getByText('Değiştirildi', { exact: true })).toBeHidden();
    const saved = await text();
    expect(saved.content).toBe(changed);
    expect(saved.revision).toBeGreaterThan(before.revision);
    await call('/api/fs/text', { path: `${root}/document.txt`, content: 'stale write', revision: before.revision }, 'PUT', 409);
    expect((await text()).content).toBe(changed);
    await editor.getByLabel('Kapat', { exact: true }).click();

    await row(page, 'document.txt').getByTitle('Sürüm Geçmişi', { exact: true }).click();
    const history = page.getByRole('dialog', { name: 'Sürüm Geçmişi', exact: true });
    await expect(history.getByText(`Rev #${before.revision}`, { exact: true })).toBeVisible();
    await screenshot(page, 'versions');
    page.once('dialog', dialog => dialog.accept());
    await history.getByRole('button', { name: 'Geri Dön', exact: true }).first().click();
    await expect.poll(async () => (await text()).content).toBe(before.content);
    const restored = await versions();
    expect(restored.revision).toBeGreaterThan(saved.revision);
    expect(restored.content.some((item: { revision: number }) => item.revision === saved.revision)).toBe(true);
    await history.getByLabel('Kapat', { exact: true }).click();
  });

  test('trash collision preserves replacement and restores original elsewhere', async ({ page }) => {
    await row(page, 'collision.txt').getByTitle('Sil', { exact: true }).click();
    const deletion = page.getByRole('dialog', { name: 'Öğeyi Sil', exact: true });
    await deletion.getByRole('button', { name: 'Sil', exact: true }).click();
    await expect(row(page, 'collision.txt')).toBeHidden();
    await upload('collision.txt', Buffer.from('Replacement content'));
    await page.getByRole('button', { name: 'Çöp Kutusu', exact: true }).click();
    const trashRow = page.locator('tbody tr').filter({ hasText: `${root}/collision.txt` });
    await expect(trashRow).toBeVisible();
    await screenshot(page, 'trash');
    await trashRow.getByTitle('Geri Yükle', { exact: true }).click();
    const collision = page.getByRole('dialog', { name: 'Geri Yükleme Çakışması', exact: true });
    await expect(collision).toBeVisible();
    await collision.getByLabel('Alternatif Geri Yükleme Yolu').fill(`${root}/restored.txt`);
    await collision.getByRole('button', { name: 'Bu Konuma Geri Yükle', exact: true }).click();
    await expect(collision).toBeHidden();
    expect((await call(`/api/fs/text?path=${encodeURIComponent(`${root}/collision.txt`)}`, undefined, 'GET')).content).toBe('Replacement content');
    expect((await call(`/api/fs/text?path=${encodeURIComponent(`${root}/restored.txt`)}`, undefined, 'GET')).content).toBe('Original trash content');

    await call('/api/fs/remove', { dir: root, names: ['restored.txt'] });
    await page.getByRole('button', { name: 'Yenile', exact: true }).click();
    const purgeRow = page.locator('tbody tr').filter({ hasText: `${root}/restored.txt` });
    await expect(purgeRow).toBeVisible();
    await purgeRow.getByTitle('Kalıcı Olarak Sil', { exact: true }).click();
    await page.getByRole('dialog', { name: 'Kalıcı Olarak Sil', exact: true }).getByRole('button', { name: 'Evet, Kalıcı Olarak Sil', exact: true }).click();
    await expect(purgeRow).toBeHidden();
  });

  test('password-protected guest preview, editable slug and expiry', async ({ page, browser }) => {
    await row(page, 'picture.png').getByTitle('Paylaş', { exact: true }).click();
    const dialog = page.getByRole('dialog', { name: 'Paylaşım Bağlantısı Oluştur', exact: true });
    const slug = `e2e-${run}`;
    shareIds.add(slug);
    await dialog.getByLabel('Özel Bağlantı Adı (Slug)').fill(slug);
    await dialog.getByLabel('Erişim Parolası (İsteğe bağlı)').fill('Test-only-share-password');
    await screenshot(page, 'share-dialog');
    await dialog.getByRole('button', { name: 'Paylaşım Oluştur', exact: true }).click();
    await expect(dialog.getByText('Paylaşım Bağlantınız Hazır')).toBeVisible();
    await dialog.getByLabel('Kapat', { exact: true }).click();

    const guest = await browser.newContext({ baseURL: process.env.DEPGO_BASE_URL || 'http://localhost:15244' });
    try {
      const guestPage = await guest.newPage();
      await guestPage.goto(`/s/${slug}`);
      await guestPage.getByPlaceholder('Parola').fill('wrong-password');
      const rejected = guestPage.waitForResponse(response => response.url().includes('/share/auth') && response.request().method() === 'POST');
      await guestPage.getByRole('button', { name: 'Erişimi Aç', exact: true }).click();
      expect((await (await rejected).json()).code).toBe(403);
      await guestPage.getByPlaceholder('Parola').fill('Test-only-share-password');
      await guestPage.getByRole('button', { name: 'Erişimi Aç', exact: true }).click();
      await expect(guestPage.getByText('Depgo Paylaşımı', { exact: true })).toBeVisible();
      await guestPage.getByTitle('Önizle', { exact: true }).click();
      const image = guestPage.getByRole('dialog', { name: 'picture.png' }).locator('img');
      await expect.poll(() => image.evaluate((element: HTMLImageElement) => element.naturalWidth)).toBe(400);
      await screenshot(guestPage, 'guest-file');

      const renamed = `${slug}-new`;
      await call('/api/share/update', { share_id: slug, new_share_id: renamed });
      shareIds.add(renamed);
      const old = await guest.request.get(`/api/public/share/info?share_id=${slug}`);
      expect((await old.json()).code).toBe(404);
      const current = await guest.request.get(`/api/public/share/info?share_id=${renamed}`);
      expect((await current.json()).code).toBe(200);
      await call('/api/share/update', { share_id: renamed, expire_at: '2000-01-01T00:00:00Z' });
      const expired = await guest.request.get(`/api/public/share/info?share_id=${renamed}`);
      expect((await expired.json()).code).toBe(410);
    } finally {
      await guest.close();
    }
  });
});
