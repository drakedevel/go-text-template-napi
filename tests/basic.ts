import {Template} from '..';

describe('Template', () => {
  test('constructor works', () => {
    expect(() => {
      new Template('test');
    }).not.toThrow();
  });
});
