import { withMermaid } from "vitepress-plugin-mermaid";
import { fileURLToPath, URL } from 'node:url'

// https://vitepress.dev/reference/site-config
export default withMermaid({
  title: "Sonic Operator",
  description: "Kubernetes Operator to manage SONiC switches",
  base: "/sonic-operator/",
  head: [['link', { rel: 'icon', href: 'https://raw.githubusercontent.com/ironcore-dev/ironcore/refs/heads/main/docs/assets/logo_borderless.svg' }]],
  vite: {
    resolve: {
      alias: [
        {
          // Override default theme footer with our funding notice.
          find: /^.*\/VPFooter\.vue$/,
          replacement: fileURLToPath(
            new URL('./theme/components/VPFooter.vue', import.meta.url)
          )
        },
      ]
    }
  },
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Concepts', link: '/concepts/overview' },
      { text: 'Quickstart', link: '/quickstart' },
      { text: 'IronCore Documentation', link: 'https://ironcore-dev.github.io' },
    ],

    editLink: {
      // Assumption: repo is published under ironcore-dev on branch main.
      pattern: 'https://github.com/ironcore-dev/sonic-operator/blob/main/docs/:path',
      text: 'Edit this page on GitHub'
    },

    logo: {
      src: 'https://raw.githubusercontent.com/ironcore-dev/ironcore/refs/heads/main/docs/assets/logo_borderless.svg',
      width: 24,
      height: 24
    },

    search: {
      provider: 'local'
    },

    sidebar: [
      {
        items: [
          { text: "Quickstart", link: '/quickstart' },
          {
            text: 'Installation',
            collapsed: true,
            items: [
              { text: 'Kustomize', link: '/installation/kustomize' },
              { text: 'Helm', link: '/installation/helm' },
            ]
          },
          { text: 'Architecture', link: '/architecture' },
          { text: 'API Reference', link: '/api-reference/api' },
        ]
      },
      {
        text: 'Concepts',
        collapsed: false,
        items: [
          { text: 'Overview', link: '/concepts/overview' },
          { text: 'Resources', link: '/concepts/resources' },
        ]
      },
      {
        text: 'Usage',
        collapsed: false,
        items: [
          { text: 'Getting started', link: '/usage/getting-started' },
          { text: 'Provisioning', link: '/usage/provisioning' },
          { text: 'Agent', link: '/usage/agent' },
        ]
      },
      {
        text: 'Developer Guide',
        collapsed: false,
        items: [
          { text: 'Dev workflow', link: '/development/dev-workflow' },
          { text: 'REUSE', link: '/development/reuse' },
          { text: 'Documentation', link: '/development/dev-docs' },
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/ironcore-dev/sonic-operator' }
    ],
  }
})
