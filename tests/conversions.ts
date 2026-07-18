import { describe, expect, jest } from '@jest/globals';
import { fc, it } from '@fast-check/jest';

import { Template } from '..';

describe('JS-Go value conversion', () => {
  // TODO: Undefined
  const arbScalar = fc.oneof(
    // TODO: re-enable when Go->JS side is implemented
    // fc.bigInt(),
    fc.boolean(),
    fc.constant(null),
    fc.double(),
    fc.string(),
  );
  // TODO: Nested arrays/objects
  const arbValue = fc.oneof(
    arbScalar,
    fc.array(arbScalar),
    fc.dictionary(fc.string(), arbScalar),
  );

  it.prop([arbValue], {
    examples: [[{ ['__proto__']: '' }]],
  })('should roundtrip', (val) => {
    const jsFn = jest.fn();
    const template = new Template('t').funcs({ jsFn }).parse('{{ jsFn .val }}');
    template.executeString({ val });
    expect(jsFn).toHaveBeenCalledWith(val);
  });
});
