// Register tsx to handle .ts files before cucumber loads step definitions.
require('tsx/cjs');

module.exports = {
  default: {
    format: ['html:reports/cucumber.html'],
    require: [
      'src/hooks.ts',
      'src/support/**/*.ts',
      'src/steps/**/*.ts',
    ],
    paths: process.argv.some(arg => arg.endsWith('.feature')) ? [] : ['../docs/e2e/journeys/*.feature'],
    language: 'en',
    timeout: 10000,
  },
};
