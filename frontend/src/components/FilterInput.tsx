import React from 'react';
import { SearchIcon, CloseIcon } from './Icons';

interface FilterInputProps {
  value: string;
  onChange: (val: string) => void;
  placeholder?: string;
}

export const FilterInput: React.FC<FilterInputProps> = ({
  value,
  onChange,
  placeholder = 'Bu klasörde filtrele...',
}) => {
  return (
    <div className="relative flex items-center w-full min-w-0 sm:max-w-xs">
      <div className="absolute left-3 text-charcoal-400 pointer-events-none">
        <SearchIcon className="w-4 h-4" />
      </div>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        aria-label="Dosyaları filtrele"
        className="w-full pl-9 pr-8 py-1.5 bg-white border border-charcoal-200 rounded-lg text-sm text-charcoal-800 placeholder-charcoal-400 focus:outline-none focus:ring-2 focus:ring-accent/30 focus:border-accent transition-colors"
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange('')}
          className="absolute right-2.5 p-0.5 text-charcoal-400 hover:text-charcoal-600 rounded"
          title="Filtreyi temizle"
        >
          <CloseIcon className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  );
};
