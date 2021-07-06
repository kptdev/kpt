// Matches a path like /book/02-concepts/01-packages
const bookPath = /^\/book\/(\d+)-(.+)\/(\d+)?-?(.+)?/;

function addGitHubWidget(hook) {
  const issueIcon = document.createElement("span");
  issueIcon.innerText = "bug_report";
  issueIcon.classList.add("material-icons-outlined");
  const createIssue = document.createElement("a");
  createIssue.target = "_blank";
  createIssue.title = "Create documentation issue";
  createIssue.appendChild(issueIcon);

  const editIcon = document.createElement("span");
  editIcon.innerText = "edit";
  editIcon.classList.add("material-icons-outlined");
  const editPage = document.createElement("a");
  editPage.target = "_blank";
  editPage.title = "Edit this page";
  editPage.appendChild(editIcon);

  // Refresh widget links.
  hook.doneEach(function () {
    createIssue.href = `https://github.com/GoogleContainerTools/kpt/issues/new?labels=documentation&title=Docs: ${document.title} (${document.})`;

    let path = document.location.pathname;
    const pageName = path.match(bookPath) ? "00.md" : "README.md";
    path += path.endsWith("/") ? pageName : ".md";
    editPage.href = `https://github.com/GoogleContainerTools/kpt/edit/main/site${path}`;

    const container = document.createElement("div");
    container.classList.add("github-widget");
    container.appendChild(createIssue);
    container.appendChild(editPage);
    document
      .getElementsByClassName("docsify-pagination-container")
      .item(0)
      .append(container);
  });
}

function convertFromHugo(content) {
  const hugoHideDirectives = /{{% hide %}}.+?{{% \/hide %}}/gms;
  const hugoDirectiveTags = /{{.*}}/g;

  content = processHugoTitleHeading(content);
  return content.replace(hugoHideDirectives, "").replace(hugoDirectiveTags, "");
}

async function addVersionDropdown() {
  const sidebar = document.getElementsByClassName("sidebar").item(0);
  const latestVersion = "v1.0.0-beta";
  const versionDropdown = `
  <div class="dropdown">
                <button class="btn btn-primary dropdown-toggle" type="button" data-toggle="dropdown">${latestVersion}
                <span class="caret"></span></button>
                <ol class="dropdown-menu">
                <li><a href="/installation/">${latestVersion}</a></li>
                <li><a href="https://googlecontainertools.github.io/kpt/installation/" target="_self">v0.39</a></li>
                </ol>
              </div>
                `;
  const node = document.createElement("div");
  node.innerHTML = versionDropdown;
  sidebar.getElementsByClassName("app-name").item(0).appendChild(node);
}

function showBookPageFooters() {
  const isBookPage = document.location.pathname
    .toLowerCase()
    .startsWith("/book");

  const hideButtonsToNonBookPages = (buttons) => {
    buttons.forEach((el) => {
      url = new URL(el.lastElementChild.href);
      el.style.display = isBookPage && url.pathname.toLowerCase().startsWith("/book")
        ? "flex"
        : "none";
    });
  };

  const previousPaginationButtons = Array.from(
    document.getElementsByClassName("pagination-item--previous")
  );

  const nextPaginationButtons = Array.from(
    document.getElementsByClassName("pagination-item--next")
  );

  hideButtonsToNonBookPages(
    previousPaginationButtons.concat(nextPaginationButtons)
  );
}

function addSidebarCollapsibility(sidebar) {
  const tocLists = Array.from(sidebar?.getElementsByTagName("ul"));

  // Hide a child list if neither its parent nor any of its descendants are active.
  tocLists.forEach((ul) =>
    ul.parentElement.classList.contains("active") ||
    ul.getElementsByClassName("active").length
      ? ul.classList.remove("inactive")
      : ul.classList.add("inactive")
  );
}

// Make Markdown standard titles (# Title) out of the following:
// +++
// title: Page Title
// +++
function processHugoTitleHeading(content) {
  const titleBlock = /^[\+\-]{3}[\s\S]*?^[\+\-]{3}$/m;
  const titleMatch = content.match(/title:\s*["'](.*)["']/);

  const titleHeading = titleMatch ? `# ${titleMatch[1]}` : "";

  return content.replace(titleBlock, titleHeading);
}

// Convert Hugo Asciinema directives to HTML.
function processAsciinemaTags(content) {
  const asciinemaDirective = /{{<\s*asciinema.+key="(.+?)".+}}/g;

  return content.replace(
    asciinemaDirective,
    (_, fileName) =>
      `<asciinema-player src="${window.location.origin}/static/casts/${fileName}.cast" cols="160"></asciinema-player>`
  );
}
// Workaround for https://github.com/docsifyjs/docsify/pull/1468
function defaultLinkTargets() {
  const externalPageLinks = Array.from(
    document.getElementsByTagName("a")
  ).filter(
    (a) =>
      window.Docsify.util.isExternal(a.href) &&
      !window.$docsify.crossOriginLinks.includes(a.href)
  );
  externalPageLinks.forEach(
    (a) => (a.target = window.$docsify.externalLinkTarget)
  );
}

function localPlugins(hook, _vm) {
  // Process Markdown directives appropriately.
  hook.beforeEach(function (content) {
    content = processAsciinemaTags(content);

    // Until all source markdown files stop using Hugo directives,
    // convert here for compatibility.
    content = convertFromHugo(content);
    return content;
  });

  hook.mounted(addVersionDropdown);

  // Show navigation footer for book pages.
  hook.doneEach(showBookPageFooters);

  // Reset all external links to their appropriate targets.
  hook.doneEach(defaultLinkTargets);

  addGitHubWidget(hook);

  // Process elements in the navigation sidebar.
  hook.doneEach(function () {
    const sidebar = document.getElementsByClassName("sidebar-nav").item(0);

    // Only show child pages for currently active page to avoid sidebar cluttering.
    addSidebarCollapsibility(sidebar);
  });
}

// Load plugins into Docsify.
window.$docsify = window.$docsify || {};
window.$docsify.plugins = [localPlugins].concat(window.$docsify.plugins || []);

