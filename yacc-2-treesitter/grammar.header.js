/// <reference types="tree-sitter-cli/dsl" />
// @ts-check

const PREC = {
  // () [] . ++ --
  postfix: 10,
  // @ * & && ~ ! + - ++ --
  prefix: 9,
  // * / % *%
  multiplicative: 8,
  // << >>
  shift: 7,
  // ^ | &
  bitwise: 6,
  // + -
  additive: 5,
  // == != >= <= > <
  comparative: 4,
  // &&
  and: 3,
  // ||
  or: 2,
  // ?:
  ternary: 1,
  // == *= /= %= *%= += -= <<= >>= &= ^= |=
  assign: 0,
};

module.exports = grammar({
  name: "c3",

  extras: ($) => [/\s|\\\r?\n/, $.comment],

  inline: ($) => [
    $._top_level_item,
    $._statement,
    $._empty_statement,
    $._declaration_statement,
    $._path,
    $._type_path,
    $._expression,
    $._initializer,
    $._literal,
    $._string_literal,
    $._raw_string_literal,
    $._multiline_string_literal,
    $._prefix_expression,
    $._postfix_expression,
    $._type,
    $._integer_type,
    $._float_type,
    $._function_signature,
    $._var_declaration,
    $._struct_declaration,
    $._field_struct_declaration,
    $._function_body,
    $._lambda_body,
  ],

  conflicts: ($) => [
    [$.compound_statement, $.initializers],
    [$.assignment_expression, $.lambda_expression],
    [$.range_expression, $.lambda_expression],
  ],

  word: ($) => $.identifier,
  rules: {
    // http://stackoverflow.com/questions/13014947/regex-to-match-a-c-style-multiline-comment/36328890#36328890
    comment: ($) =>
        token(
            choice(
                seq("//", /(\\(.|\r?\n)|[^\\\n])*/),
                seq("/*", /[^*]*\*+([^/*][^*]*\*+)*/, "/")
            )
        ),
