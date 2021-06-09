/**
 * @jest-environment jsdom
 */

const plugins = require("./plugins");

test("clipboard copying button exists only on shell blocks with a '$'", () => {
  document.body.innerHTML = `<pre v-pre="" data-lang="shell" class="language-shell">
    <code class="lang-shell language-shell">
    No copy button because $.
    </code>
    </pre>
    <pre v-pre="" data-lang="shell" class="language-shell">
    <code class="lang-shell language-shell">
    $ kpt fn <span class="token builtin class-name">eval</span> wordpress --image gcr.io/kpt-fn/set-namespace:v0.1 -- <span class="token assign-left variable">namespace</span><span class="token operator">=</span>mywordpress
    </code>
    </pre>
    <pre v-pre="" data-lang="yaml" class="language-yaml">
    <code class="lang-yaml language-yaml">
    $ No copy button because language.
    </code>
    </pre>`;
  plugins.addCodeCopyButtons();
  const preTags = document.getElementsByTagName("pre");
  expect(preTags.item(0).getElementsByClassName("copy-button").length).toBe(0);
  expect(preTags.item(1).getElementsByClassName("copy-button").length).toBe(1);
  expect(preTags.item(2).getElementsByClassName("copy-button").length).toBe(0);
});
