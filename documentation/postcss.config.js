// Local postcss config, adjacent to package.json/node_modules so that
// postcss-load-config resolves plugins from the project's own
// node_modules/ and does not try to walk out of Hugo's module cache
// (which Node 24's permission model blocks).
//
// Mirrors Docsy 0.14's upstream postcss.config.js (autoprefixer only).
module.exports = {
  plugins: {
    autoprefixer: {},
  },
};
