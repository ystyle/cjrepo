import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'CJRepo',
  description: '仓颉语言私有包仓库',
  base: process.env.DOCS_BASE || '/docs/',
  lang: 'zh-CN',
  ignoreDeadLinks: true,
  outDir: '../dist/docs',
  themeConfig: {
    nav: [
      { text: '首页', link: '/' },
      { text: '指南', link: '/guide/' },
      { text: 'API', link: '/api/' },
      { text: '部署', link: '/deploy/' },
    ],
    sidebar: {
      '/guide/': [
        { text: '快速开始', link: '/guide/' },
        { text: '用户管理', link: '/guide/users' },
        { text: '团队权限', link: '/guide/teams' },
        { text: '上游代理', link: '/guide/upstream' },
        { text: '常见问题', link: '/guide/faq' },
      ],
      '/deploy/': [
        { text: '部署指南', link: '/deploy/' },
        { text: 'Docker', link: '/deploy/docker' },
        { text: '环境变量', link: '/deploy/env' },
      ],
      '/api/': [
        { text: '概述', link: '/api/' },
      ],
      '/deploy/': [
        { text: '部署指南', link: '/deploy/' },
        { text: 'Docker', link: '/deploy/docker' },
        { text: '环境变量', link: '/deploy/env' },
      ],
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/ystyle/cjrepo' },
    ],
    footer: {
      message: '基于 MIT 协议开源',
      copyright: 'Copyright © 2026 CJRepo',
    },
  },
})
