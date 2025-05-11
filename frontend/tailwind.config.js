/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/**/*.{js,jsx,ts,tsx}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        surface: {
          DEFAULT: "hsl(var(--surface))",
          foreground: "hsl(var(--surface-foreground))",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      typography: {
        DEFAULT: {
          css: {
            color: 'hsl(var(--foreground))',
            a: {
              color: 'hsl(var(--primary))',
              '&:hover': {
                color: 'hsl(var(--primary))',
              },
            },
            'h1,h2,h3,h4,h5,h6': {
              color: 'hsl(var(--foreground))',
            },
            code: {
              color: 'hsl(var(--foreground))',
              background: 'hsl(var(--muted))',
              padding: '0.2em 0.4em',
              borderRadius: '0.25rem',
            },
            'code::before': {
              content: '""',
            },
            'code::after': {
              content: '""',
            },
            pre: {
              backgroundColor: 'hsl(var(--background))',
              color: 'hsl(var(--foreground))',
            },
            blockquote: {
              borderLeftColor: 'hsl(var(--muted))',
              color: 'hsl(var(--muted-foreground))',
            },
            hr: {
              borderColor: 'hsl(var(--border))',
            },
            'ul > li::before': {
              backgroundColor: 'hsl(var(--muted))',
            },
            strong: {
              color: 'hsl(var(--foreground))',
            },
            thead: {
              color: 'hsl(var(--foreground))',
              borderBottomColor: 'hsl(var(--border))',
            },
            tbody: {
              tr: {
                borderBottomColor: 'hsl(var(--border))',
              },
            },
          },
        },
      },
    },
  },
  plugins: [
    require('tailwind-scrollbar')({ nocompatible: true }),
    require('@tailwindcss/typography'),
  ],
}
