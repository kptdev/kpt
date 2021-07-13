// Add title to book Markdown pages based on directory structure.
function processBookPageTitle(content) {
  const pathname = window.location.pathname.toLowerCase();

  const bookPathMatch = pathname.match(/^\/book\/(\d+)-(.+)\/(\d+)?-?(.+)?/);

  if (content && !content.startsWith("<!DOCTYPE html>") && bookPathMatch) {
    const pageNumber = parseInt(bookPathMatch[3]);

    // Use chapter name if on intro page and page name otherwise.
    const chapterNum = `# ${parseInt(bookPathMatch[1])}${
      pageNumber > 0 ? `.${pageNumber}` : ""
    }`;
    const pageTitle = pageNumber > 0 ? bookPathMatch[4] : bookPathMatch[2];

    content =
      `${chapterNum} ${pageTitle.replace(/-/g, " ").toTitleCase()}\n` +
      content;
  }

  return content;
}

// Load plugins into Docsify.
window.$docsify = window.$docsify || {};
window.$docsify.plugins = [
  (hook, _vm) => hook.beforeEach(processBookPageTitle),
].concat(window.$docsify.plugins || []);

// Export functions for testing.
if (typeof module !== "undefined") {
  module.exports = { processBookPageTitle };
}
