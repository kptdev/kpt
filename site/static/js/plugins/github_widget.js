// Matches a path like /book/02-concepts/01-packages
const bookPath = /^\/book\/(\d+)-(.+)\/(\d+)?-?(.+)?/;

const issueIcon = document.createElement("span");
issueIcon.innerText = "bug_report";
issueIcon.classList.add("material-icons-outlined");
const createIssue = document.createElement("a");
createIssue.id = "create_issue_button";
createIssue.target = "_blank";
createIssue.title = "Create documentation issue";
createIssue.appendChild(issueIcon);

const editIcon = document.createElement("span");
editIcon.innerText = "edit";
editIcon.classList.add("material-icons-outlined");
const editPage = document.createElement("a");
editPage.id = "edit_page_button";
editPage.target = "_blank";
editPage.title = "Edit this page";
editPage.appendChild(editIcon);

function addGitHubWidget() {
  createIssue.href = `https://github.com/GoogleContainerTools/kpt/issues/new?labels=documentation&title=Docs: ${document.title} (${window.location})`;

  let path = window.location.pathname;
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
}

// Load plugins into Docsify.
window.$docsify = window.$docsify || {};
window.$docsify.plugins = [].concat(function (hook, _vm) {
  hook.doneEach(addGitHubWidget);
}, window.$docsify.plugins);

// Export functions for testing.
if (typeof module !== "undefined") {
  module.exports = { addGitHubWidget };
}
