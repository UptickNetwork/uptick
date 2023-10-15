module.exports = {
  theme: 'cosmos',
  title: 'Uptick Documentation',
  locales: {
    '/': {
      lang: 'en-US'
    },
  },
  markdown: {
    extendMarkdown: (md) => {
      md.use(require("markdown-it-katex"));
    },
  },
  head: [
    [
      "link",
      {
        rel: "stylesheet",
        href:
          "https://cdnjs.cloudflare.com/ajax/libs/KaTeX/0.5.1/katex.min.css",
      },
    ],
    [
      "link",
      {
        rel: "stylesheet",
        href:
          "https://cdn.jsdelivr.net/github-markdown-css/2.2.1/github-markdown.css",
      },
    ],
  ],
  base: process.env.VUEPRESS_BASE || '/',
  plugins: [
    'vuepress-plugin-element-tabs'
  ],
  head: [
    // ['link', { rel: "apple-touch-icon", sizes: "180x180", href: "/apple-touch-icon.png" }],
    ['link', { rel: "icon", type: "image/png", sizes: "32x32", href: "/favicon32.png" }],
    ['link', { rel: "icon", type: "image/png", sizes: "16x16", href: "/favicon16.png" }],
    ['link', { rel: "manifest", href: "/site.webmanifest" }],
    ['meta', { name: "msapplication-TileColor", content: "#2e3148" }],
    ['meta', { name: "theme-color", content: "#ffffff" }],
    ['link', { rel: "icon", type: "image/svg+xml", href: "/favicon.svg" }],
    // ['link', { rel: "apple-touch-icon-precomposed", href: "/apple-touch-icon-precomposed.png" }],
  ],
  themeConfig: {
    repo: 'UptickNetwork/uptick',
    docsRepo: 'UptickNetwork/uptick',
    docsBranch: 'main',
    docsDir: 'docs',
    editLinks: true,
    custom: true,
    project: {
      name: 'Uptick',
      denom: 'Uptick',
      ticker: 'UPTICK',
      binary: 'uptickd',
      testnet_denom: 'Uptick',
      testnet_ticker: 'UPTICK',
      rpc_url: 'http://localhost:8545/',
      rpc_url_testnet: 'https://peer1.testnet.uptick.network:8645/',
      rpc_url_local: 'http://localhost:8545/',
      chain_id: '未定',
      testnet_chain_id: '7000',
      latest_version: 'v0.2.4',
      version_number: '1',
      testnet_version_number: '2',
      cosmos_block_explorer_url: 'https://explorer.testnet.uptick.network/',
      block_explorer_url:'https://evm-explorer.testnet.uptick.network'
    },
    logo: {
      src: '/uptick-black.svg',
    },
    // algolia: {
    //   id: 'BH4D9OD16A',
    //   key: 'a5d55fe5f540cc3bd28fa2c72f2b5bd8',
    //   index: 'uptick'
    // },
    topbar: {
      banner: false
    },
    sidebar: {
      auto: false,
      nav: [
        {
          title: 'Reference',
          children: [
            {
              title: 'Introduction',
              directory: true,
              path: '/intro'
            },
            {
              title: 'Quick Start',
              directory: true,
              path: '/quickstart'
            },
            {
              title: 'Basics',
              directory: true,
              path: '/basics'
            },
            {
              title: 'Core Concepts',
              directory: true,
              path: '/core'
            },
          ]
        },
        {
          title: 'Guides',
          children: [
            {
              title: 'Localnet',
              directory: true,
              path: '/guides/localnet'
            },
            {
              title: 'Keys and Wallets',
              directory: true,
              path: '/guides/keys-wallets'
            },
            {
              title: 'Ethereum Tooling',
              directory: true,
              path: '/guides/tools'
            },
            {
              title: 'Validators',
              directory: true,
              path: '/guides/validators'
            },
            {
              title: 'Upgrades',
              directory: true,
              path: '/guides/upgrades'
            },
            {
              title: 'Key Management System',
              directory: true,
              path: '/guides/kms'
            },
            {
              title: 'State sync',
              directory: true,
              path: '/guides/statesync'
            },
          ]
        },
        {
          title: 'APIs',
          children: [
            {
              title: 'JSON-RPC',
              directory: true,
              path: '/api/json-rpc'
            },
            {
              title: 'Protobuf Reference',
              directory: false,
              path: '/api/proto-docs'
            },
          ]
        },
        {
          title: 'Testnet',
          children: [
            {
              title: 'Join Testnet',
              directory: false,
              path: '/testnet/join'
            },
            {
              title: 'Token Faucet',
              directory: false,
              path: '/testnet/faucet'
            },
            {
              title: 'Deploy Node on Cloud',
              directory: false,
              path: '/testnet/cloud_providers'
            }
          ]
        },
        {
          title: 'Specifications',
          children: [{
            title: 'Modules',
            directory: true,
            path: '/modules'
          }]
        },
        {
          title: 'Block Explorers',
          children: [
            {
              title: 'Uptick (Cosmos)',
              path: 'https://explorer.testnet.uptick.network'
            },
            {
              title: 'GN (Cosmos)',
              path: 'https://uptick.explorers.guru'
            },
            {
              title: 'Blockscout (EVM)',
              path: 'https://evm-explorer.testnet.uptick.network/'
            },
          ]
        },
        {
          title: 'Resources',
          children: [
            {
              title: 'Uptick API Reference',
              path: 'https://pkg.go.dev/github.com/UptickNetwork/uptick'
            },
            {
              title: 'Ethermint Library API Reference',
              path: 'https://pkg.go.dev/github.com/tharsis/ethermint'
            },
            {
              title: 'JSON-RPC API Reference',
              path: '/api/json-rpc/endpoints'
            }
          ]
        }
      ]
    },
    gutter: {
      title: 'Help & Support',
      chat: {
        title: 'Developer Chat',
        text: 'Chat with Uptick developers on Discord.',
        url: 'https://discord.gg/8w4GUUUH39',
        bg: 'linear-gradient(103.75deg, #1B1E36 0%, #22253F 100%)'
      },
      forum: {
        title: 'Uptick Medium',
        text: 'Join the Uptick Network Medium to learn more.',
        url: 'https://uptickproject.medium.com',
        bg: 'linear-gradient(221.79deg, #3D6B99 -1.08%, #336699 95.88%)',
        logo: 'ethereum-white'
      },
      github: {
        title: 'Found an Issue?',
        text: 'Help us improve this page by suggesting edits on GitHub.',
        bg: '#F8F9FC'
      }
    },
    footer: {
      logo: '/uptick-black.svg',
      textLink: {
        text: 'uptick.network',
        url: 'https://www.uptick.network'
      },
      services: [
        {
          service: 'github',
          url: 'https://github.com/UptickNetwork/uptick'
        },
        {
          service: "twitter",
          url: "https://twitter.com/uptickproject",
        },
        {
          service: "telegram",
          url: "https://t.me/uptickproject",
        },
        {
          service: "medium",
          url: "https://uptickproject.medium.com",
        },
      ],
      smallprint: 'This website is maintained by UptickNetwork.',
      links: [{
        title: 'Documentation',
        children: [{
          title: 'Cosmos SDK Docs',
          url: 'https://docs.cosmos.network/master/'
        },
        {
          title: 'Ethereum Docs',
          url: 'https://ethereum.org/developers'
        },
        {
          title: 'Tendermint Core Docs',
          url: 'https://docs.tendermint.com'
        }
        ]
      },
      {
        title: 'Community',
        children: [{
          title: 'Uptick Community',
          url: 'https://discord.gg/8w4GUUUH39'
        },
        ]
      }
      ]
    },
    versions: [
      {
        "label": "main",
        "key": "main"
      },
    ],
  }
};