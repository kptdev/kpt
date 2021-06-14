/**
 * @jest-environment jsdom
 */

const plugins = require("./book_page_title");
require("@gouch/to-title-case");

test.each([
  ["1 The Chapter Title", "book/01-the-chapter-title/"],
  ["1.1 First Title", "book/01-anything/01-first-title"],
  [
    "1.2 A Much Longer Page Title",
    "book/01-anything/02-a-much-longer-page-title",
  ],
])("title is correct on book pages", (expectedTitle, path) => {
  delete window.location;
  window.location = new URL(path, "https://test.test");
  const transformedContent = plugins.processBookPageTitle(
    "Placeholder content"
  );
  expect(transformedContent.split("\n")[0]).toBe(`# ${expectedTitle}`);
});
