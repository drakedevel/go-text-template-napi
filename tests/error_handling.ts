import * as binding from '..';
import { Template } from '..';

const NO_FILE_ERR =
  process.platform === 'win32'
    ? 'cannot find the path specified'
    : 'no such file or directory';

describe('Template', () => {
  let template: Template;

  beforeEach(() => {
    template = new Template('test_template');
  });

  test('constructor handles incorrect argument types', () => {
    // @ts-expect-error: testing bad arguments
    expect(() => new Template(0)).toThrow('A string was expected');
  });

  test('#delims handles incorrect argument types', () => {
    // @ts-expect-error: testing bad arguments
    expect(() => template.delims(0, '')).toThrow('A string was expected');
    // @ts-expect-error: testing bad arguments
    expect(() => template.delims('', 0)).toThrow('A string was expected');
  });

  const unaryMethods = [
    'executeTemplateString',
    'lookup',
    'new',
    'option',
    'parse',
    'parseFiles',
    'parseGlob',
  ] as const;
  test.each(unaryMethods)('#%s handles incorrect argument types', (name) => {
    // @ts-expect-error: testing bad arguments
    expect(() => template[name](0)).toThrow('A string was expected');
  });

  const staticMethods = ['parseFiles', 'parseGlob'] as const;
  test.each(staticMethods)(
    'static .%s handles incorrect argument types',
    (name) => {
      // @ts-expect-error: testing bad arguments
      expect(() => Template[name](0)).toThrow('A string was expected');
    },
  );

  describe('#executeString', () => {
    it('handles unsupported value types', () => {
      expect(() => template.executeString(Symbol())).toThrow(
        'Unsupported value type',
      );
      expect(() => template.executeString([Symbol()])).toThrow(
        'Unsupported value type',
      );
      expect(() => template.executeString({ a: Symbol() })).toThrow(
        'Unsupported value type',
      );
    });

    it('propagates errors from property accesses', () => {
      const err = new Error();
      const badObj = {};
      Object.defineProperty(badObj, 'poison', {
        enumerable: true,
        get() {
          throw err;
        },
      });
      expect(() => template.executeString(badObj)).toThrow(err);
      const badArr: unknown[] = [];
      Object.defineProperty(badArr, 0, {
        enumerable: true,
        get() {
          throw err;
        },
      });
      expect(() => template.executeString(badArr)).toThrow(err);
      const badProxy = new Proxy(
        {},
        {
          ownKeys() {
            throw err;
          },
        },
      );
      expect(() => template.executeString(badProxy)).toThrow(err);
    });
  });

  describe('#executeTemplateString', () => {
    it('handles unsupported value types', () => {
      expect(() => template.executeTemplateString('', Symbol())).toThrow(
        'Unsupported value type',
      );
    });

    it('handles invalid template names', () => {
      expect(() => template.executeTemplateString('invalid')).toThrow(
        'no template "invalid"',
      );
    });
  });

  describe('#funcs', () => {
    it('captures panics', () => {
      expect(() =>
        template.funcs({ ['']() {} }),
      ).toThrowErrorMatchingInlineSnapshot(
        `"caught panic: function name "" is not a valid identifier"`,
      );
    });

    it('ignores undefined functions', () => {
      template.funcs({
        myFunc() {
          return 'hello';
        },
        // @ts-expect-error: testing invalid args
        myUndef: undefined,
      });
      expect(template.parse('{{ myFunc }}').executeString()).toBe('hello');
      expect(() => template.parse('{{ myUndef }}')).toThrow(
        'function "myUndef" not defined',
      );
    });

    it('handles incorrect argument types', () => {
      // @ts-expect-error: testing invalid args
      expect(() => template.funcs(null)).toThrow(
        'Cannot convert undefined or null to object',
      );
    });

    it('handles invalid function types', () => {
      expect(() =>
        // @ts-expect-error: testing invalid args
        template.funcs({ invalid: 42 }),
      ).toThrowErrorMatchingInlineSnapshot(`"Key 'invalid' is not a function"`);
    });

    it('handles un-mappable Go types', () => {
      const myFunc = jest.fn();
      template.addSprigFuncs().funcs({ myFunc }).parse('{{ myFunc 123i }}');
      expect(() => template.executeString()).toThrow(
        "can't convert Go value of type complex128",
      );
      template.parse('{{ myFunc (dict "key" 123i) }}');
      expect(() => template.executeString()).toThrow(
        "can't convert Go value of type complex128",
      );
      template.parse('{{ myFunc (list 123i) }}');
      expect(() => template.executeString()).toThrow(
        "can't convert Go value of type complex128",
      );
      expect(myFunc).not.toHaveBeenCalled();
    });

    it('propagates errors from property accesses', () => {
      const err = new Error();
      const funcs = {};
      Object.defineProperty(funcs, 'poison', {
        enumerable: true,
        get() {
          throw err;
        },
      });
      expect(() => template.funcs(funcs)).toThrow(err);
    });
  });

  test('#parseFiles propagates errors', () => {
    expect(() => template.parseFiles('/invalid/path/to/template/file')).toThrow(
      NO_FILE_ERR,
    );
  });

  test('#parseGlob propagates errors', () => {
    expect(() => template.parseGlob('/invalid/path/to/template/dir/*')).toThrow(
      'pattern matches no files',
    );
  });

  test('static .parseFiles propagates errors', () => {
    expect(() => Template.parseFiles('/invalid/path/to/template/file')).toThrow(
      NO_FILE_ERR,
    );
  });

  test('static .parseGlob propagates errors', () => {
    expect(() => Template.parseGlob('/invalid/path/to/template/dir/*')).toThrow(
      'pattern matches no files',
    );
  });

  test('methods handle missing arguments', () => {
    // @ts-expect-error: testing missing argument
    expect(() => template.parse()).toThrow('A string was expected');
  });

  test('methods handle invalid this value', () => {
    // @ts-expect-error: testing missing argument
    const unwrapped = new Template();
    expect(() => unwrapped.parse('')).toThrow('missing or invalid type tag');
  });

  test('static methods handle missing arguments', () => {
    // @ts-expect-error: testing missing argument
    expect(() => Template.parseGlob()).toThrow('A string was expected');
  });
});

test('helpers handle unsupported value types', () => {
  expect(() => binding.htmlEscaper(Symbol())).toThrow('Unsupported value type');
});
