import type { Config } from 'tailwindcss';

export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],

  theme: {
    extend: {
      // Carbon's g90 palette, mirrored so Tailwind utilities speak the same colors as
      // the components. carbon-components-svelte compiles theme colors into g90.css
      // literally rather than exposing --cds-* custom properties, so there is nothing
      // to reference at runtime and the values have to be duplicated here.
      colors: {
        carbon: {
          bg: '#262626', // page background
          layer: '#393939', // ui-01 / field-01, e.g. Tile
          'layer-hover': '#4c4c4c', // hover-ui, what Carbon uses for table rows
          border: '#525252', // ui-03
          text: '#f4f4f4', // text-01
          'text-secondary': '#c6c6c6', // text-02
          'text-helper': '#8d8d8d', // text-03
          error: '#ff8389',
          success: '#42be65',
          link: '#78a9ff'
        }
      }
    }
  },

  plugins: []
} satisfies Config;
