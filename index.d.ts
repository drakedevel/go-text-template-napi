type FuncMap = {[name: string]: (...args: unknown[]) => unknown};

export class Template {
  constructor(name: string);

  clone(): Template;
  definedTemplates(): string;
  delims(left: string, right: string): Template;
  execute(data: unknown): string;
  executeTemplate(name: string, data: unknown): string;
  funcs(funcMap: FuncMap): Template;
  lookup(name: string): Template | undefined;
  name(): string;
  new(name: string): Template;
  option(...opts: string[]): Template;
  parse(text: string): Template;
  parseFiles(...files: string[]): Template;
  parseGlob(glob: string): Template;
  templates(): Template[];

  static parseFiles(...files: string[]): Template;
  static parseGlob(glob: string): Template;
}

export function htmlEscapeString(str: string): string;
export function htmlEscaper(...args: unknown[]): string;
export function jsEscapeString(str: string): string;
export function jsEscaper(...args: unknown[]): string;
export function urlQueryEscaper(...args: unknown[]): string;
