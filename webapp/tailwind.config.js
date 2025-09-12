/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx,svelte}",
  ],
  theme: {
    extend: {
      fontFamily: {
        'sans': ['Inter', 'ui-sans-serif', 'system-ui', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'Noto Sans', 'sans-serif'],
      },
      colors: {
        'nb-bg': '#fffbea',
        'nb-fg': '#0f172a',
        'nb-accent': '#111827',
        'nb-primary': '#ff6b35',
        'nb-secondary': '#f7931e',
        'nb-success': '#4ecdc4',
        'nb-danger': '#ff5757',
        'nb-warning': '#ffd93d',
        'nb-info': '#6c5ce7',
        'nb-surface': '#ffffff',
        'nb-surface-alt': '#f8fafc',
        'nb-muted': '#e2e8f0',
        'nb-text': '#0f172a',
        'nb-text-muted': '#64748b',
        'nb-text-light': '#94a3b8',
      },
      boxShadow: {
        'nb': '8px 8px 0 var(--nb-accent)',
        'nb-sm': '4px 4px 0 var(--nb-accent)',
        'nb-lg': '12px 12px 0 var(--nb-accent)',
        'nb-xl': '16px 16px 0 var(--nb-accent)',
      },
    },
  },
  plugins: [],
}
