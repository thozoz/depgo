import { describe, it, expect } from 'vitest';
import { joinPath, splitPath, getBreadcrumbs } from '../utils/format';

describe('Path handling utilities', () => {
  describe('joinPath', () => {
    it('normalizes single root path', () => {
      expect(joinPath('/')).toBe('/');
      expect(joinPath('')).toBe('/');
    });

    it('joins multiple segments safely', () => {
      expect(joinPath('/', 'documents', 'test.pdf')).toBe('/documents/test.pdf');
      expect(joinPath('documents', 'photos')).toBe('/documents/photos');
    });

    it('handles backslashes and redundant slashes', () => {
      expect(joinPath('\\windows\\path', 'file.txt')).toBe('/windows/path/file.txt');
      expect(joinPath('//folder//', '//subfolder//')).toBe('/folder/subfolder');
    });

    it('resolves dot segments securely', () => {
      expect(joinPath('/a/b', '..', 'c')).toBe('/a/c');
      expect(joinPath('/a', '.', 'b')).toBe('/a/b');
      expect(joinPath('/', '..', '..')).toBe('/');
    });
  });

  describe('splitPath', () => {
    it('splits file path into directory and filename', () => {
      expect(splitPath('/docs/report.pdf')).toEqual({ dir: '/docs', name: 'report.pdf' });
      expect(splitPath('/file.txt')).toEqual({ dir: '/', name: 'file.txt' });
    });

    it('handles root directory', () => {
      expect(splitPath('/')).toEqual({ dir: '/', name: '' });
    });
  });

  describe('getBreadcrumbs', () => {
    it('returns root breadcrumb for /', () => {
      const crumbs = getBreadcrumbs('/');
      expect(crumbs).toEqual([{ name: 'Kök Dizin', path: '/' }]);
    });

    it('returns hierarchical breadcrumbs for nested paths', () => {
      const crumbs = getBreadcrumbs('/a/b/c');
      expect(crumbs).toEqual([
        { name: 'Kök Dizin', path: '/' },
        { name: 'a', path: '/a' },
        { name: 'b', path: '/a/b' },
        { name: 'c', path: '/a/b/c' },
      ]);
    });
  });
});
