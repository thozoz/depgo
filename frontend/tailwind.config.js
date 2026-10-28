/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        charcoal: {
          50: '#F4F4F5',
          100: '#E4E4E7',
          200: '#D4D4D8',
          300: '#A1A1AA',
          400: '#71717A',
          500: '#52525B',
          600: '#3F3F46',
          700: '#27272A',
          800: '#18181B',
          900: '#09090B',
        },
        warm: {
          50: '#FCFBF9',
          100: '#F7F6F2',
          200: '#EFECE6',
        },
        accent: {
          DEFAULT: '#10B981',
          hover: '#059669',
          light: '#D1FAE5',
          dark: '#047857',
        },
      },
    },
  },
  plugins: [],
};
