/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        surface: {
          DEFAULT: '#121214',
          raised: '#1c1c1f',
          hover: '#252528',
        },
        accent: {
          DEFAULT: '#fa2d48',
          muted: '#c4243a',
        },
      },
      fontFamily: {
        sans: ['Segoe UI', 'system-ui', 'sans-serif'],
      },
      keyframes: {
        'status-in': {
          from: { opacity: '0', transform: 'translateY(-6px) scale(0.97)' },
          to: { opacity: '1', transform: 'translateY(0) scale(1)' },
        },
      },
      animation: {
        'status-in': 'status-in 0.32s cubic-bezier(0.2, 0, 0, 1)',
      },
      transitionTimingFunction: {
        apple: 'cubic-bezier(0.2, 0, 0, 1)',
      },
      maxWidth: {
        content: '72rem', // 1152px — main tab column on desktop
      },
    },
  },
  plugins: [],
}
