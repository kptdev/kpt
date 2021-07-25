function addCodeCopyButtons() {
  const preBlocks = Array.from(document.getElementsByTagName("pre")).filter(
    (el) =>
      el.classList.contains("language-shell") &&
      el.firstElementChild.textContent
        .split("\n")
        .find((line) => line.trimLeft().startsWith("$"))
  );

  const makeButton = () => {
    const copyButton = document.createElement("button");
    const buttonClassName = "copy-button";
    copyButton.classList.add(buttonClassName);
    copyButton.title = "Copy to clipboard";

    const copyIcon = document.createElement("span");
    copyIcon.innerText = "copy";
    copyIcon.classList.add("material-icons-outlined");
    copyButton.appendChild(copyIcon);

    copyButton.addEventListener("click", (el) =>
      navigator.clipboard.writeText([
        el
          .composedPath()
          .find((el) => el.classList.contains(buttonClassName))
          .previousElementSibling.textContent.split("\n")
          .map((s) => s.trim())
          .filter(
            (s, ix, arr) =>
              s.startsWith("$") || (ix > 0 && arr[ix - 1].endsWith("\\"))
          )
          .map((s) => s.replace(/^\$\s+/, ""))
          .join("\n"),
      ])
    );
    return copyButton;
  };
  preBlocks.forEach((pre) => pre.appendChild(makeButton()));
}

// Load plugins into Docsify.
window.$docsify = window.$docsify || {};
window.$docsify.plugins = [].concat(function (hook, _vm) {
  hook.doneEach(addCodeCopyButtons);
}, window.$docsify.plugins);

// Export functions for testing.
if (typeof module !== "undefined") {
  module.exports = { addCodeCopyButtons };
}
