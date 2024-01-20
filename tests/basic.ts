import * as path from 'path';

import * as binding from '..';
import {Template} from '..';

const templateDir = path.join(__dirname, 'data');

describe('Template', () => {
  let template: Template;

  beforeEach(() => {
    template = new Template('test_template');
  });

  describe('#clone', () => {
    it('works', () => {
      const cloned = template.parse('results').clone();
      expect(cloned).not.toBe(template);
      expect(cloned.name()).toBe(template.name());
      expect(cloned.executeString()).toBe('results');
    });

    it('works with JS functions', () => {
      const f1 = jest.fn(() => 'result1');
      const f2 = jest.fn(() => 'result2');
      const f2alt = jest.fn(() => 'result2-alt');
      const cloned = template.funcs({ f1, f2 }).parse('{{ f1 }} {{ f2 }}').clone();
      cloned.funcs({f2: f2alt});
      expect(template.executeString()).toBe('result1 result2');
      expect(cloned.executeString()).toBe('result1 result2-alt');
      expect(f1).toHaveBeenCalledTimes(2);
      expect(f2).toHaveBeenCalledTimes(1);
      expect(f2alt).toHaveBeenCalledTimes(1);
    });
  });

  test('#definedTemplates works', () => {
    expect(template.definedTemplates()).toBe('');
    template.parse('{{define "foo"}}{{ end }}');
    expect(template.definedTemplates()).toMatch(/^; defined templates are: ("foo", "test_template"|"test_template", "foo")$/);
  });

  test('#delims works', () => {
    template.delims('<<', '>>').parse('<< . >>');
    expect(template.executeString('hello')).toBe('hello');
  });

  test('#executeString works', () => {
    template.parse('{{ .foo }}, {{ .bar }}');
    expect(template.executeString({foo: 'hello', bar: 'world'})).toBe('hello, world');
  });

  test('#executeTemplateString works', () => {
    template.parse('{{ define "inner" }}inner {{ .param }}{{ end }}outer');
    expect(template.executeTemplateString('inner', {param: 'param'})).toBe('inner param');
  });

  describe('#funcs', () => {
    it('works', () => {
      const myFunc = jest.fn((value: string) => `pre-${value}-post`);
      template.funcs({ myFunc });
      template.parse('{{ myFunc "hello" }}');
      expect(template.executeString()).toBe('pre-hello-post');
      expect(myFunc).toHaveBeenCalledWith('hello');
      expect(myFunc).toHaveBeenCalledTimes(1);
    });

    it('can overwrite old functions', () => {
      const oldFunc = jest.fn();
      template.funcs({myFunc: oldFunc});
      const newFunc = () => 'ok';
      template.funcs({myFunc: newFunc}).parse('{{ myFunc }}');
      expect(template.executeString()).toBe('ok');
      expect(oldFunc).not.toHaveBeenCalled();
    });

    it('round-trips JS types', () => {
      const myFunc = jest.fn((value: unknown) => value);
      template.funcs({ myFunc });
      template.parse('{{ myFunc . }}');
      // TODO: undefined handling
      expect(template.executeString(null)).toBe('<no value>');
      expect(myFunc).toHaveBeenLastCalledWith(null);
      expect(template.executeString(true)).toBe('true');
      expect(myFunc).toHaveBeenLastCalledWith(true);
      expect(template.executeString(123)).toBe('123');
      expect(myFunc).toHaveBeenLastCalledWith(123);
      expect(template.executeString('param')).toBe('param');
      expect(myFunc).toHaveBeenLastCalledWith('param');
      expect(template.executeString({foo: 'bar'})).toBe('map[foo:bar]');
      expect(myFunc).toHaveBeenLastCalledWith({foo: 'bar'});
      expect(template.executeString(['foo', 'bar'])).toBe('[foo bar]');
      expect(myFunc).toHaveBeenLastCalledWith(['foo', 'bar']);
      // TODO: BigInt support when supported
    });

    it('passes Go integers to JS', () => {
      const myFunc = jest.fn((value: unknown) => `${value}`);
      template.funcs({ myFunc });
      template.parse('{{ myFunc 123 }}');
      expect(template.executeString()).toBe('123');
      expect(myFunc).toHaveBeenCalledWith(123);
    })

    it('propagates JS exceptions', () => {
      const err = new Error('test error');
      template.funcs({ throwErr() { throw err; } });
      template.parse('{{ throwErr }}');
      expect(() => template.executeString()).toThrow(err);
    });

    test('native functions can overwrite JS functions', () => {
      const oldAtoi = jest.fn();
      template.funcs({ atoi: oldAtoi }).addSprigFuncs().parse('{{ atoi "0" }}');
      expect(template.executeString()).toBe('0');
      expect(oldAtoi).not.toHaveBeenCalled();
    });

    test('JS functions can overwrite native functions', () => {
      const atoi = () => "ok";
      template.addSprigFuncs().funcs({ atoi }).parse('{{ atoi "0" }}');
      expect(template.executeString()).toBe('ok');
    });
  });

  describe('#lookup', () => {
    it('works', () => {
      template.parse('{{ define "foo" }}{{ end }}');
      const fooTemplate = template.lookup('foo');
      expect(fooTemplate).toBeInstanceOf(Template);
      expect(fooTemplate?.name()).toBe('foo');
    });

    it('returns undefined on invalid templates', () => {
      expect(template.lookup('invalid')).toBeUndefined();
    });
  });

  test('#name works', () => {
    expect(template.name()).toBe('test_template');
  });

  test('#new works', () => {
    const inner = template.new('inner').parse('new {{ .param }}');
    expect(inner).toBeInstanceOf(Template);
    expect(inner.name()).toBe('inner');
    expect(template.executeTemplateString('inner', {param: 'param'})).toBe('new param');
  });

  describe('#option', () => {
    it('works', () => {
      template.parse('{{ .param }}');
      expect(template.executeString({})).toBe('<no value>');
      template.option('missingkey=error');
      expect(() => template.executeString({})).toThrowErrorMatchingInlineSnapshot(`"template: test_template:1:3: executing "test_template" at <.param>: map has no entry for key "param""`);
    });

    it('captures panics', () => {
      expect(() => template.option('invalidArg')).toThrowErrorMatchingInlineSnapshot(`"caught panic: unrecognized option: invalidArg"`);
    });
  });

  test('#parseFiles works', () => {
    template.parseFiles(path.join(templateDir, 'a.tpl'));
    expect(template.executeTemplateString('a.tpl')).toBe('template a\n');
  });

  test('#parseGlob works', () => {
    template.parseGlob(path.join(templateDir, '*.tpl'));
    expect(template.executeTemplateString('a.tpl')).toBe('template a\n');
    expect(template.executeTemplateString('b.tpl')).toBe('template b\n');
  });

  test('#templates works', () => {
    expect(template.templates()).toStrictEqual([]);
    template.new('foo').parse('foo contents');
    const templates = template.templates();
    expect(templates.length).toBe(1);
    expect(templates[0]?.name()).toBe('foo');
    expect(templates[0]?.executeString()).toBe('foo contents');
  });

  test('#addSprigFuncs works', () => {
    template.addSprigFuncs().parse('{{ dict "a" 42 }}');
    expect(template.executeString()).toBe('map[a:42]');
  });

  test('#addSprigHermeticFuncs works', () => {
    template.addSprigHermeticFuncs().parse('{{ dict "a" 42 }}');
    expect(template.executeString()).toBe('map[a:42]');
    expect(() => template.parse('{{ uuidv4 }}')).toThrowErrorMatchingInlineSnapshot(`"template: test_template:1: function "uuidv4" not defined"`);
  });

  test('static .parseFiles works', () => {
    const parsed = Template.parseFiles(path.join(templateDir, 'a.tpl'));
    expect(parsed.name()).toBe('a.tpl');
    expect(parsed.executeString()).toBe('template a\n');
  });

  test('static .parseGlob works', () => {
    const parsed = Template.parseGlob(path.join(templateDir, '*.tpl'));
    expect(parsed.name()).toBe('a.tpl');
    expect(parsed.executeString()).toBe('template a\n');
    expect(parsed.executeTemplateString('b.tpl')).toBe('template b\n');
  });

  test('JS BigInt support works', () => {
    template.parse('{{ . }}');
    const value = (1n << 128n) + 2n; // Endianness test with 64-bit words
    expect(template.executeString(value)).toBe(value.toString());
    expect(template.executeString(-value)).toBe((-value).toString());
  })
});

test('htmlEscapeString works', () => {
  expect(binding.htmlEscapeString('<br>')).toBe('&lt;br&gt;');
});

test('htmlEscaper works', () => {
  expect(binding.htmlEscaper('foo', '<bar>')).toBe('foo&lt;bar&gt;');
});

test('jsEscapeString works', () => {
  expect(binding.jsEscapeString('"foo"')).toBe('\\"foo\\"');
});

test('jsEscaper works', () => {
  expect(binding.jsEscaper('foo', '"bar"')).toBe('foo\\"bar\\"');
});

test('urlQueryEscaper works', () => {
  expect(binding.urlQueryEscaper('foo', '&bar')).toBe('foo%26bar');
});
