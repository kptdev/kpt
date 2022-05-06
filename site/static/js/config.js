window.$docsify = {
  name: `<img src="/static/images/logo.png" alt="kpt" />`,
  nameLink: window.location.origin,
  search: {
    maxAge: 43200000,
    placeholder: "Search",
    paths: "auto",
  },
  loadSidebar: "sidebar.md",
  alias: {
    "/*.*/sidebar.md": "/sidebar.md",
  },
  coverpage: "coverpage.md",
  crossOriginLinks: [
    "https://catalog.kpt.dev/",
    "https://googlecontainertools.github.io/kpt/installation/",
  ],
  auto2top: true,
  repo: "true",
  pagination: {
    previousText: 'PREV',
    nextText: 'NEXT',
    crossChapter: true
  },
  corner: {
    url: "https://github.com/GoogleContainerTools/kpt",
    icon: "github",
  },
  routerMode: "history",
  markdown: {
    renderer: {
      image: function (href, title) {
        return `<img src="${href}" data-origin="${href}" alt="${title}">`;
      },
    },
  },
};
