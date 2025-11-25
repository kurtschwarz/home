import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Kurt\'s Homelab Docs',
  tagline: '',
  favicon: 'img/favicon.ico',

  future: {
    v4: true,
  },

  url: 'http://localhost:3000',
  baseUrl: '/',

  organizationName: 'kurtschwarz',
  projectName: 'homelab-docs',

  onBrokenLinks: 'throw',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          path: '../content',
          sidebarPath: './sidebars.ts',
        },
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    colorMode: {
      respectPrefersColorScheme: true,
    },
    navbar: {
      title: 'Kurt\'s Homelab Docs',
      items: [],
    },
    footer: {
      style: 'dark',
      links: [],
      copyright: 'Built with <a href="https://docusaurus.io/" target="_blank">Docusaurus</a>.',
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
}

export default config
