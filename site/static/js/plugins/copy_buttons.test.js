/**
 * @jest-environment jsdom
 */

const plugins = require("./copy_buttons");

const originalClipboard = { ...global.navigator.clipboard };
beforeEach(() => {
  const mockClipboard = {
    writeText: jest.fn(),
  };
  global.navigator.clipboard = mockClipboard;
});

afterEach(() => {
  jest.resetAllMocks();
  global.navigator.clipboard = originalClipboard;
});

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

test.each([
  [
    [
      `kpt fn source wordpress \\
| kpt fn eval - --image gcr.io/kpt-fn/set-namespace:v0.1 -- namespace=mywordpress \\
| kpt fn eval - --image gcr.io/kpt-fn/set-labels:v0.1 -- app=wordpress env=prod \\
| kpt fn sink wordpress`,
    ],
    `$ kpt fn source wordpress \\
| kpt fn eval - --image gcr.io/kpt-fn/set-namespace:v0.1 -- namespace=mywordpress \\
| kpt fn eval - --image gcr.io/kpt-fn/set-labels:v0.1 -- app=wordpress env=prod \\
| kpt fn sink wordpress`,
  ],
  [
    [
      `kpt fn eval - --image \\
gcr.io/kpt-fn/set-namespace:v0.1 -- namespace=mywordpress`,
    ],
    `$ kpt fn eval - --image \\
gcr.io/kpt-fn/set-namespace:v0.1 -- namespace=mywordpress
output`,
  ],
])(
  "clipboard copying button copies all lines of command: %s",
  (expected, shellCodeText) => {
    document.body.innerHTML = `<pre v-pre="" data-lang="shell" class="language-shell">
    <code class="lang-shell language-shell">
    ${shellCodeText}
    </code>
    </pre>`;
    plugins.addCodeCopyButtons();
    document.getElementsByClassName("copy-button").item(0).click();
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(expected);
  }
);
