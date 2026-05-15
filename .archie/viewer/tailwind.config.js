/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
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
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
        ink: {
          DEFAULT: '#023047',
          50: '#a9e1fd', 100: '#a9e1fd', 200: '#54c3fb', 300: '#06a3f1',
          400: '#04699b', 500: '#023047', 600: '#012638', 700: '#011c2a',
          800: '#01131c', 900: '#00090e', 950: '#000507',
        },
        teal: {
          DEFAULT: '#219ebc',
          50: '#ceeef6', 100: '#ceeef6', 200: '#9cddee', 300: '#6bcce5',
          400: '#39bcdc', 500: '#219ebc', 600: '#1a7d95', 700: '#145d70',
          800: '#0d3e4b', 900: '#071f25', 950: '#040f13',
        },
        papaya: {
          DEFAULT: '#8ecae6',
          50: '#e8f4fa', 100: '#e8f4fa', 200: '#d2eaf5', 300: '#bbdff0',
          400: '#a5d5eb', 500: '#8ecae6', 600: '#51aed9', 700: '#288ab7',
          800: '#1b5c7a', 900: '#0d2e3d', 950: '#07171f',
        },
        tangerine: {
          DEFAULT: '#ffb703',
          50: '#fff1cd', 100: '#fff1cd', 200: '#ffe39b', 300: '#ffd569',
          400: '#ffc637', 500: '#ffb703', 600: '#d09500', 700: '#9c7000',
          800: '#684b00', 900: '#342500', 950: '#1a1300',
        },
        brandy: {
          DEFAULT: '#fb8500',
          50: '#ffe7cb', 100: '#ffe7cb', 200: '#ffce97', 300: '#ffb663',
          400: '#ff9e2f', 500: '#fb8500', 600: '#c86b00', 700: '#965000',
          800: '#643500', 900: '#321b00', 950: '#190e00',
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      keyframes: {
        "accordion-down": { from: { height: 0 }, to: { height: "var(--radix-accordion-content-height)" } },
        "accordion-up": { from: { height: "var(--radix-accordion-content-height)" }, to: { height: 0 } },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
      },
      typography: {
        DEFAULT: {
          css: {
            '--tw-prose-body': '#023047',
            '--tw-prose-headings': '#023047',
            '--tw-prose-lead': '#1a7d95',
            '--tw-prose-links': '#219ebc',
            '--tw-prose-bold': '#023047',
            '--tw-prose-counters': '#219ebc',
            '--tw-prose-bullets': '#219ebc',
            '--tw-prose-hr': '#8ecae6',
            '--tw-prose-quotes': '#023047',
            '--tw-prose-quote-borders': '#219ebc',
            '--tw-prose-captions': '#1a7d95',
            '--tw-prose-code': '#219ebc',
            'code:not(pre code)': {
              backgroundColor: 'rgba(142,202,230,0.15)',
              borderRadius: '0.25rem',
              padding: '0.15em 0.4em',
              fontSize: '0.875em',
              fontWeight: '600',
            },
            'code::before': { content: 'none' },
            'code::after': { content: 'none' },
            '--tw-prose-pre-code': '#8ecae6',
            '--tw-prose-pre-bg': '#023047',
            '--tw-prose-th-borders': '#8ecae6',
            '--tw-prose-td-borders': '#8ecae6',
          },
        },
      },
    },
  },
  plugins: [
    require("tailwindcss-animate"),
    require('@tailwindcss/typography'),
  ],
}
