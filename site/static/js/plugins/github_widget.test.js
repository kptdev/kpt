/**
 * @jest-environment jsdom
 */

const plugins = require("./github_widget");

test.each([
  [
    "https://github.com/GoogleContainerTools/kpt/issues/new?labels=documentation&title=Docs:%20title%20(https://test.test/book/01-the-chapter-title/)",
    "https://github.com/GoogleContainerTools/kpt/edit/main/site/book/01-the-chapter-title/00.md",
    "book/01-the-chapter-title/",
  ],
  [
    "https://github.com/GoogleContainerTools/kpt/issues/new?labels=documentation&title=Docs:%20title%20(https://test.test/book/01-the-chapter-title/05-page.md)",
    "https://github.com/GoogleContainerTools/kpt/edit/main/site/book/01-the-chapter-title/05-page.md.md",
    "book/01-the-chapter-title/05-page.md",
  ],
  [
    "https://github.com/GoogleContainerTools/kpt/issues/new?labels=documentation&title=Docs:%20title%20(https://test.test/faq/)",
    "https://github.com/GoogleContainerTools/kpt/edit/main/site/faq/README.md",
    "faq/",
  ],
])("github urls are correct", (expectedIssueUrl, expectedEditUrl, path) => {
  // Configure test environment.
  delete window.location;
  window.location = new URL(path, "https://test.test");
  document.title = "title";
  const container = document.createElement("div");
  container.classList.add("docsify-pagination-container");
  
  document.body.append(container);
  plugins.addGitHubWidget();
  const issueUrl = document.getElementById("create_issue_button").href;
  const editUrl = document.getElementById("edit_page_button").href;
  expect(issueUrl).toBe(expectedIssueUrl);
  expect(editUrl).toBe(expectedEditUrl);
});
