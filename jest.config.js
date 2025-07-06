const { createDefaultPreset } = require('ts-jest');
module.exports = {
  ...createDefaultPreset(),
  testMatch: ['**/tests/**/*.ts'],
};
