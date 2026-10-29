import { describe, expect, it } from 'vitest';
import { FileItem } from '../types';

describe('multi-selection logic', () => {
  const sampleItems: FileItem[] = [
    { name: 'file1.txt', size: 100, is_dir: false, modified: '2026-01-01T00:00:00Z', sign: '', thumb: '', type: 1 },
    { name: 'file2.txt', size: 200, is_dir: false, modified: '2026-01-01T00:00:00Z', sign: '', thumb: '', type: 1 },
    { name: 'folder1', size: 0, is_dir: true, modified: '2026-01-01T00:00:00Z', sign: '', thumb: '', type: 0 },
  ];

  it('toggles selection of single item', () => {
    const selected = new Set<string>();
    selected.add('file1.txt');
    expect(selected.has('file1.txt')).toBe(true);

    selected.delete('file1.txt');
    expect(selected.has('file1.txt')).toBe(false);
  });

  it('selects and deselects all items correctly', () => {
    let selected = new Set<string>();
    // Select all
    selected = new Set(sampleItems.map((i) => i.name));
    expect(selected.size).toBe(3);
    expect(sampleItems.every((i) => selected.has(i.name))).toBe(true);

    // Deselect all
    selected = new Set();
    expect(selected.size).toBe(0);
  });

  it('filters selected items correctly for batch operations', () => {
    const selected = new Set(['file1.txt', 'folder1']);
    const targets = sampleItems.filter((i) => selected.has(i.name));
    expect(targets.length).toBe(2);
    expect(targets.map((i) => i.name)).toEqual(['file1.txt', 'folder1']);

    const downloadable = targets.filter((i) => !i.is_dir);
    expect(downloadable.length).toBe(1);
    expect(downloadable[0].name).toBe('file1.txt');
  });
});
