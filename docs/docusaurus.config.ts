import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

const config: Config = {
  title: "bx2cloud",
  tagline: "Documentation for bx2cloud, the local cloud provider",
  favicon: "img/favicon.ico",

  future: {
    v4: true,
  },

  url: "https://benasb.github.io",
  baseUrl: "/bx2cloud/",

  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      {
        pages: false,
        docs: {
          routeBasePath: "/",
          sidebarPath: "./sidebars.ts",
          editUrl: "https://github.com/benasb/bx2cloud/tree/main/docs/",
        },
        blog: false,
        theme: {
          customCss: "./src/css/custom.css",
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    image: "img/social-card.png",
    navbar: {
      title: "Cloud",
      logo: {
        alt: "Bx2 logo",
        src: "img/logo-white.svg",
        srcDark: "img/logo-black.svg",
      },
      items: [
        {
          href: "https://github.com/BenasB/bx2cloud",
          label: "GitHub",
          position: "right",
        },
      ],
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
