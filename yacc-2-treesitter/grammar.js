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
        path: $ => choice(
            seq(
                $.IDENT,
                $.SCOPE,
            ),
            seq(
                $.path,
                $.IDENT,
                $.SCOPE,
            ),
        ),

        path_const: $ => choice(
            seq(
                $.path,
                $.CONST_IDENT,
            ),
            $.CONST_IDENT,
        ),

        path_ident: $ => choice(
            seq(
                $.path,
                $.IDENT,
            ),
            $.IDENT,
        ),

        path_at_ident: $ => choice(
            seq(
                $.path,
                $.AT_IDENT,
            ),
            $.AT_IDENT,
        ),

        ident_expr: $ => choice(
            $.CONST_IDENT,
            $.IDENT,
            $.AT_IDENT,
        ),

        local_ident_expr: $ => choice(
            $.CT_IDENT,
            $.HASH_IDENT,
        ),

        ct_call: $ => choice(
            $.CT_ALIGNOF,
            $.CT_EXTNAMEOF,
            $.CT_NAMEOF,
            $.CT_OFFSETOF,
            $.CT_QNAMEOF,
        ),

        ct_castable: $ => $.CT_ASSIGNABLE,
        ct_analyse: $ => choice(
            $.CT_EVAL,
            $.CT_DEFINED,
            $.CT_SIZEOF,
            $.CT_STRINGIFY,
            $.CT_IS_CONST,
        ),

        ct_arg: $ => choice(
            $.CT_VACONST,
            $.CT_VAARG,
            $.CT_VAREF,
            $.CT_VAEXPR,
        ),

        flat_path: $ => choice(
            seq(
                $.primary_expr,
                $.param_path,
            ),
            $.type,
            $.primary_expr,
        ),

        maybe_optional_type: $ => choice(
            $.optional_type,
            $.empty,
        ),

        string_expr: $ => choice(
            $.STRING_LITERAL,
            seq(
                $.string_expr,
                $.STRING_LITERAL,
            ),
        ),

        bytes_expr: $ => choice(
            $.BYTES,
            seq(
                $.bytes_expr,
                $.BYTES,
            ),
        ),

        expr_block: $ => seq(
            $.LBRAPIPE,
            $.opt_stmt_list,
            $.RBRAPIPE,
        ),

        base_expr: $ => choice(
            $.string_expr,
            $.INTEGER,
            $.bytes_expr,
            $.NUL,
            seq(
                $.BUILTIN,
                $.CONST_IDENT,
            ),
            seq(
                $.BUILTIN,
                $.IDENT,
            ),
            $.CHAR_LITERAL,
            $.REAL,
            $.TRUE,
            $.FALSE,
            seq(
                $.path,
                $.ident_expr,
            ),
            $.ident_expr,
            $.local_ident_expr,
            seq(
                $.type,
                $.initializer_list,
            ),
            seq(
                $.type,
                '.',
                $.access_ident,
            ),
            seq(
                $.type,
                '.',
                $.CONST_IDENT,
            ),
            seq(
                '(',
                $.expr,
                ')',
            ),
            $.expr_block,
            seq(
                $.ct_call,
                '(',
                $.flat_path,
                ')',
            ),
            seq(
                $.ct_arg,
                '(',
                $.expr,
                ')',
            ),
            seq(
                $.ct_analyse,
                '(',
                $.expression_list,
                ')',
            ),
            $.CT_VACOUNT,
            seq(
                $.CT_FEATURE,
                '(',
                $.CONST_IDENT,
                ')',
            ),
            seq(
                $.CT_AND,
                '(',
                $.expression_list,
                ')',
            ),
            seq(
                $.ct_castable,
                '(',
                $.expr,
                ',',
                $.type,
                ')',
            ),
            seq(
                $.lambda_decl,
                $.compound_statement,
            ),
        ),

        primary_expr: $ => choice(
            $.base_expr,
            $.initializer_list,
        ),

        range_loc: $ => choice(
            $.expr,
            seq(
                '^',
                $.expr,
            ),
        ),

        range_expr: $ => choice(
            seq(
                $.range_loc,
                $.DOTDOT,
                $.range_loc,
            ),
            seq(
                $.range_loc,
                $.DOTDOT,
            ),
            seq(
                $.DOTDOT,
                $.range_loc,
            ),
            seq(
                $.range_loc,
                ':',
                $.range_loc,
            ),
            seq(
                ':',
                $.range_loc,
            ),
            seq(
                $.range_loc,
                ':',
            ),
            $.DOTDOT,
        ),

        call_inline_attributes: $ => choice(
            $.AT_IDENT,
            seq(
                $.call_inline_attributes,
                $.AT_IDENT,
            ),
        ),

        call_invocation: $ => choice(
            seq(
                '(',
                $.call_arg_list,
                ')',
            ),
            seq(
                '(',
                $.call_arg_list,
                ')',
                $.call_inline_attributes,
            ),
        ),

        access_ident: $ => choice(
            $.IDENT,
            $.AT_IDENT,
            $.HASH_IDENT,
            seq(
                $.CT_EVAL,
                '(',
                $.expr,
                ')',
            ),
            $.TYPEID,
        ),

        call_trailing: $ => choice(
            seq(
                '[',
                $.range_loc,
                ']',
            ),
            seq(
                '[',
                $.range_expr,
                ']',
            ),
            $.call_invocation,
            seq(
                $.call_invocation,
                $.compound_statement,
            ),
            seq(
                '.',
                $.access_ident,
            ),
            $.generic_expr,
            $.INC_OP,
            $.DEC_OP,
            '!',
            $.BANGBANG,
        ),

        call_stmt_expr: $ => choice(
            $.base_expr,
            seq(
                $.call_stmt_expr,
                $.call_trailing,
            ),
        ),

        call_expr: $ => choice(
            $.primary_expr,
            seq(
                $.call_expr,
                $.call_trailing,
            ),
        ),

        unary_expr: $ => choice(
            $.call_expr,
            seq(
                $.unary_op,
                $.unary_expr,
            ),
        ),

        unary_stmt_expr: $ => choice(
            $.call_stmt_expr,
            seq(
                $.unary_op,
                $.unary_expr,
            ),
        ),

        unary_op: $ => choice(
            '&',
            $.AND_OP,
            '*',
            '+',
            '-',
            '~',
            '!',
            $.INC_OP,
            $.DEC_OP,
            seq(
                '(',
                $.type,
                ')',
            ),
        ),

        mult_op: $ => choice(
            '*',
            '/',
            '%',
        ),

        mult_expr: $ => choice(
            $.unary_expr,
            seq(
                $.mult_expr,
                $.mult_op,
                $.unary_expr,
            ),
        ),

        mult_stmt_expr: $ => choice(
            $.unary_stmt_expr,
            seq(
                $.mult_stmt_expr,
                $.mult_op,
                $.unary_expr,
            ),
        ),

        shift_op: $ => choice(
            $.SHL_OP,
            $.SHR_OP,
        ),

        shift_expr: $ => choice(
            $.mult_expr,
            seq(
                $.shift_expr,
                $.shift_op,
                $.mult_expr,
            ),
        ),

        shift_stmt_expr: $ => choice(
            $.mult_stmt_expr,
            seq(
                $.shift_stmt_expr,
                $.shift_op,
                $.mult_expr,
            ),
        ),

        bit_op: $ => choice(
            '&',
            '^',
            '|',
        ),

        bit_expr: $ => choice(
            $.shift_expr,
            seq(
                $.bit_expr,
                $.bit_op,
                $.shift_expr,
            ),
        ),

        bit_stmt_expr: $ => choice(
            $.shift_stmt_expr,
            seq(
                $.bit_stmt_expr,
                $.bit_op,
                $.shift_expr,
            ),
        ),

        additive_op: $ => choice(
            '+',
            '-',
        ),

        additive_expr: $ => choice(
            $.bit_expr,
            seq(
                $.additive_expr,
                $.additive_op,
                $.bit_expr,
            ),
        ),

        additive_stmt_expr: $ => choice(
            $.bit_stmt_expr,
            seq(
                $.additive_stmt_expr,
                $.additive_op,
                $.bit_expr,
            ),
        ),

        relational_op: $ => choice(
            '<',
            '>',
            $.LE_OP,
            $.GE_OP,
            $.EQ_OP,
            $.NE_OP,
        ),

        relational_expr: $ => choice(
            $.additive_expr,
            seq(
                $.relational_expr,
                $.relational_op,
                $.additive_expr,
            ),
        ),

        relational_stmt_expr: $ => choice(
            $.additive_stmt_expr,
            seq(
                $.relational_stmt_expr,
                $.relational_op,
                $.additive_expr,
            ),
        ),

        rel_or_lambda_expr: $ => choice(
            $.relational_expr,
            seq(
                $.lambda_decl,
                $.IMPLIES,
                $.relational_expr,
            ),
        ),

        and_expr: $ => choice(
            $.relational_expr,
            seq(
                $.and_expr,
                $.AND_OP,
                $.relational_expr,
            ),
        ),

        and_stmt_expr: $ => choice(
            $.relational_stmt_expr,
            seq(
                $.and_stmt_expr,
                $.AND_OP,
                $.relational_expr,
            ),
        ),

        or_expr: $ => choice(
            $.and_expr,
            seq(
                $.or_expr,
                $.OR_OP,
                $.and_expr,
            ),
        ),

        or_stmt_expr: $ => choice(
            $.and_stmt_expr,
            seq(
                $.or_stmt_expr,
                $.OR_OP,
                $.and_expr,
            ),
        ),

        suffix_expr: $ => choice(
            $.or_expr,
            seq(
                $.or_expr,
                '?',
            ),
            seq(
                $.or_expr,
                '?',
                '!',
            ),
        ),

        suffix_stmt_expr: $ => choice(
            $.or_stmt_expr,
            seq(
                $.or_stmt_expr,
                '?',
            ),
            seq(
                $.or_stmt_expr,
                '?',
                '!',
            ),
        ),

        ternary_expr: $ => choice(
            $.suffix_expr,
            seq(
                $.or_expr,
                '?',
                $.expr,
                ':',
                $.ternary_expr,
            ),
            seq(
                $.suffix_expr,
                $.ELVIS,
                $.ternary_expr,
            ),
            seq(
                $.suffix_expr,
                $.OPTELSE,
                $.ternary_expr,
            ),
            seq(
                $.lambda_decl,
                $.implies_body,
            ),
        ),

        ternary_stmt_expr: $ => choice(
            $.suffix_stmt_expr,
            seq(
                $.or_stmt_expr,
                '?',
                $.expr,
                ':',
                $.ternary_expr,
            ),
            seq(
                $.suffix_stmt_expr,
                $.ELVIS,
                $.ternary_expr,
            ),
            seq(
                $.suffix_stmt_expr,
                $.OPTELSE,
                $.ternary_expr,
            ),
            seq(
                $.lambda_decl,
                $.implies_body,
            ),
        ),

        assignment_op: $ => choice(
            '=',
            $.ADD_ASSIGN,
            $.SUB_ASSIGN,
            $.MUL_ASSIGN,
            $.DIV_ASSIGN,
            $.MOD_ASSIGN,
            $.SHL_ASSIGN,
            $.SHR_ASSIGN,
            $.AND_ASSIGN,
            $.XOR_ASSIGN,
            $.OR_ASSIGN,
        ),

        empty: $ => optional(seq()),
        assignment_expr: $ => choice(
            $.ternary_expr,
            seq(
                $.CT_TYPE_IDENT,
                '=',
                $.type,
            ),
            seq(
                $.unary_expr,
                $.assignment_op,
                $.assignment_expr,
            ),
        ),

        assignment_stmt_expr: $ => choice(
            $.ternary_stmt_expr,
            seq(
                $.CT_TYPE_IDENT,
                '=',
                $.type,
            ),
            seq(
                $.unary_stmt_expr,
                $.assignment_op,
                $.assignment_expr,
            ),
        ),

        implies_body: $ => seq(
            $.IMPLIES,
            $.expr,
        ),

        lambda_decl: $ => seq(
            $.FN,
            $.maybe_optional_type,
            $.fn_parameter_list,
            $.opt_attributes,
        ),

        expr_no_list: $ => $.assignment_stmt_expr,
        expr: $ => $.assignment_expr,
        constant_expr: $ => $.ternary_expr,
        param_path_element: $ => choice(
            seq(
                '[',
                $.expr,
                ']',
            ),
            seq(
                '[',
                $.expr,
                $.DOTDOT,
                $.expr,
                ']',
            ),
            seq(
                '.',
                $.primary_expr,
            ),
        ),

        param_path: $ => choice(
            $.param_path_element,
            seq(
                $.param_path,
                $.param_path_element,
            ),
        ),

        arg: $ => choice(
            seq(
                $.param_path,
                '=',
                $.expr,
            ),
            $.type,
            seq(
                $.param_path,
                '=',
                $.type,
            ),
            $.expr,
            seq(
                $.CT_VASPLAT,
                '(',
                $.range_expr,
                ')',
            ),
            seq(
                $.CT_VASPLAT,
                '(',
                ')',
            ),
            seq(
                $.ELLIPSIS,
                $.expr,
            ),
        ),

        arg_list: $ => choice(
            $.arg,
            seq(
                $.arg_list,
                ',',
                $.arg,
            ),
        ),

        call_arg_list: $ => choice(
            $.arg_list,
            seq(
                $.arg_list,
                ';',
            ),
            seq(
                $.arg_list,
                ';',
                $.parameters,
            ),
            ';',
            seq(
                ';',
                $.parameters,
            ),
            $.empty,
        ),

        opt_arg_list_trailing: $ => choice(
            $.arg_list,
            seq(
                $.arg_list,
                ',',
            ),
            $.empty,
        ),

        interfaces: $ => choice(
            seq(
                $.TYPE_IDENT,
                $.opt_generic_parameters,
            ),
            seq(
                $.interfaces,
                ',',
                $.TYPE_IDENT,
                $.opt_generic_parameters,
            ),
        ),

        opt_interface_impl: $ => choice(
            seq(
                '(',
                $.interfaces,
                ')',
            ),
            seq(
                '(',
                ')',
            ),
            $.empty,
        ),

        enum_constants: $ => choice(
            $.enum_constant,
            seq(
                $.enum_constants,
                ',',
                $.enum_constant,
            ),
        ),

        enum_list: $ => choice(
            $.enum_constants,
            seq(
                $.enum_constants,
                ',',
            ),
        ),

        enum_constant: $ => choice(
            seq(
                $.CONST_IDENT,
                $.opt_attributes,
            ),
            seq(
                $.CONST_IDENT,
                '(',
                $.arg_list,
                ')',
                $.opt_attributes,
            ),
            seq(
                $.CONST_IDENT,
                '(',
                $.arg_list,
                ',',
                ')',
                $.opt_attributes,
            ),
        ),

        identifier_list: $ => choice(
            $.IDENT,
            seq(
                $.identifier_list,
                ',',
                $.IDENT,
            ),
        ),

        enum_param_decl: $ => choice(
            $.type,
            seq(
                $.type,
                $.IDENT,
            ),
            seq(
                $.type,
                $.IDENT,
                '=',
                $.expr,
            ),
        ),

        base_type: $ => choice(
            $.VOID,
            $.BOOL,
            $.CHAR,
            $.ICHAR,
            $.SHORT,
            $.USHORT,
            $.INT,
            $.UINT,
            $.LONG,
            $.ULONG,
            $.INT128,
            $.UINT128,
            $.FLOAT,
            $.DOUBLE,
            $.FLOAT16,
            $.BFLOAT16,
            $.FLOAT128,
            $.IPTR,
            $.UPTR,
            $.ISZ,
            $.USZ,
            $.ANYFAULT,
            $.ANY,
            $.TYPEID,
            seq(
                $.TYPE_IDENT,
                $.opt_generic_parameters,
            ),
            seq(
                $.path,
                $.TYPE_IDENT,
                $.opt_generic_parameters,
            ),
            $.CT_TYPE_IDENT,
            seq(
                $.CT_TYPEOF,
                '(',
                $.expr,
                ')',
            ),
            seq(
                $.CT_TYPEFROM,
                '(',
                $.constant_expr,
                ')',
            ),
            seq(
                $.CT_VATYPE,
                '(',
                $.constant_expr,
                ')',
            ),
            seq(
                $.CT_EVALTYPE,
                '(',
                $.constant_expr,
                ')',
            ),
        ),

        type: $ => choice(
            $.base_type,
            seq(
                $.type,
                '*',
            ),
            seq(
                $.type,
                '[',
                $.constant_expr,
                ']',
            ),
            seq(
                $.type,
                '[',
                ']',
            ),
            seq(
                $.type,
                '[',
                '*',
                ']',
            ),
            seq(
                $.type,
                $.LVEC,
                $.constant_expr,
                $.RVEC,
            ),
            seq(
                $.type,
                $.LVEC,
                '*',
                $.RVEC,
            ),
        ),

        optional_type: $ => choice(
            $.type,
            seq(
                $.type,
                '!',
            ),
        ),

        local_decl_after_type: $ => choice(
            $.CT_IDENT,
            seq(
                $.CT_IDENT,
                '=',
                $.constant_expr,
            ),
            seq(
                $.IDENT,
                $.opt_attributes,
            ),
            seq(
                $.IDENT,
                $.opt_attributes,
                '=',
                $.expr,
            ),
        ),

        local_decl_storage: $ => choice(
            $.STATIC,
            $.TLOCAL,
        ),

        decl_or_expr: $ => choice(
            $.var_decl,
            seq(
                $.optional_type,
                $.local_decl_after_type,
            ),
            $.expr,
        ),

        var_decl: $ => choice(
            seq(
                $.VAR,
                $.IDENT,
                '=',
                $.expr,
            ),
            seq(
                $.VAR,
                $.CT_IDENT,
                '=',
                $.expr,
            ),
            seq(
                $.VAR,
                $.CT_IDENT,
            ),
            seq(
                $.VAR,
                $.CT_TYPE_IDENT,
                '=',
                $.type,
            ),
            seq(
                $.VAR,
                $.CT_TYPE_IDENT,
            ),
        ),

        initializer_list: $ => seq(
            '{',
            $.opt_arg_list_trailing,
            '}',
        ),

        ct_case_stmt: $ => choice(
            seq(
                $.CT_CASE,
                $.constant_expr,
                ':',
                $.opt_stmt_list,
            ),
            seq(
                $.CT_CASE,
                $.type,
                ':',
                $.opt_stmt_list,
            ),
            seq(
                $.CT_DEFAULT,
                ':',
                $.opt_stmt_list,
            ),
        ),

        ct_switch_body: $ => choice(
            $.ct_case_stmt,
            seq(
                $.ct_switch_body,
                $.ct_case_stmt,
            ),
        ),

        ct_for_stmt: $ => seq(
            $.CT_FOR,
            '(',
            $.for_cond,
            ')',
            $.opt_stmt_list,
            $.CT_ENDFOR,
        ),

        ct_foreach_stmt: $ => choice(
            seq(
                $.CT_FOREACH,
                '(',
                $.CT_IDENT,
                ':',
                $.expr,
                ')',
                $.opt_stmt_list,
                $.CT_ENDFOREACH,
            ),
            seq(
                $.CT_FOREACH,
                '(',
                $.CT_IDENT,
                ',',
                $.CT_IDENT,
                ':',
                $.expr,
                ')',
                $.opt_stmt_list,
                $.CT_ENDFOREACH,
            ),
        ),

        ct_switch: $ => choice(
            seq(
                $.CT_SWITCH,
                '(',
                $.constant_expr,
                ')',
            ),
            seq(
                $.CT_SWITCH,
                '(',
                $.type,
                ')',
            ),
            $.CT_SWITCH,
        ),

        ct_switch_stmt: $ => seq(
            $.ct_switch,
            $.ct_switch_body,
            $.CT_ENDSWITCH,
        ),

        var_stmt: $ => seq(
            $.var_decl,
            ';',
            $.decl_stmt_after_type,
        ),

        declaration_stmt: $ => choice(
            $.const_declaration,
            seq(
                $.local_decl_storage,
                $.optional_type,
                $.decl_stmt_after_type,
                ';',
            ),
            seq(
                $.optional_type,
                $.decl_stmt_after_type,
                ';',
            ),
        ),

        return_stmt: $ => choice(
            seq(
                $.RETURN,
                $.expr,
                ';',
            ),
            seq(
                $.RETURN,
                ';',
            ),
        ),

        catch_unwrap_list: $ => choice(
            $.relational_expr,
            seq(
                $.catch_unwrap_list,
                ',',
                $.relational_expr,
            ),
        ),

        catch_unwrap: $ => choice(
            seq(
                $.CATCH,
                $.catch_unwrap_list,
            ),
            seq(
                $.CATCH,
                $.IDENT,
                '=',
                $.catch_unwrap_list,
            ),
            seq(
                $.CATCH,
                $.type,
                $.IDENT,
                '=',
                $.catch_unwrap_list,
            ),
        ),

        try_unwrap: $ => choice(
            seq(
                $.TRY,
                $.rel_or_lambda_expr,
            ),
            seq(
                $.TRY,
                $.IDENT,
                '=',
                $.rel_or_lambda_expr,
            ),
            seq(
                $.TRY,
                $.type,
                $.IDENT,
                '=',
                $.rel_or_lambda_expr,
            ),
        ),

        try_unwrap_chain: $ => choice(
            $.try_unwrap,
            seq(
                $.try_unwrap_chain,
                $.AND_OP,
                $.try_unwrap,
            ),
            seq(
                $.try_unwrap_chain,
                $.AND_OP,
                $.rel_or_lambda_expr,
            ),
        ),

        default_stmt: $ => seq(
            $.DEFAULT,
            ':',
            $.opt_stmt_list,
        ),

        case_stmt: $ => choice(
            seq(
                $.CASE,
                $.expr,
                ':',
                $.opt_stmt_list,
            ),
            seq(
                $.CASE,
                $.expr,
                $.DOTDOT,
                $.expr,
                ':',
                $.opt_stmt_list,
            ),
            seq(
                $.CASE,
                $.type,
                ':',
                $.opt_stmt_list,
            ),
        ),

        switch_body: $ => choice(
            $.case_stmt,
            $.default_stmt,
            seq(
                $.switch_body,
                $.case_stmt,
            ),
            seq(
                $.switch_body,
                $.default_stmt,
            ),
        ),

        cond_repeat: $ => choice(
            $.decl_or_expr,
            seq(
                $.cond_repeat,
                ',',
                $.decl_or_expr,
            ),
        ),

        cond: $ => choice(
            $.try_unwrap_chain,
            $.catch_unwrap,
            $.cond_repeat,
            seq(
                $.cond_repeat,
                ',',
                $.try_unwrap_chain,
            ),
            seq(
                $.cond_repeat,
                ',',
                $.catch_unwrap,
            ),
        ),

        else_part: $ => choice(
            seq(
                $.ELSE,
                $.if_stmt,
            ),
            seq(
                $.ELSE,
                $.compound_statement,
            ),
        ),

        if_stmt: $ => choice(
            seq(
                $.IF,
                $.optional_label,
                $.paren_cond,
                '{',
                $.switch_body,
                '}',
            ),
            seq(
                $.IF,
                $.optional_label,
                $.paren_cond,
                '{',
                $.switch_body,
                '}',
                $.else_part,
            ),
            seq(
                $.IF,
                $.optional_label,
                $.paren_cond,
                $.statement,
            ),
            seq(
                $.IF,
                $.optional_label,
                $.paren_cond,
                $.compound_statement,
                $.else_part,
            ),
        ),

        expr_list_eos: $ => choice(
            seq(
                $.expression_list,
                ';',
            ),
            ';',
        ),

        cond_eos: $ => choice(
            seq(
                $.cond,
                ';',
            ),
            ';',
        ),

        for_cond: $ => choice(
            seq(
                $.expr_list_eos,
                $.cond_eos,
                $.expression_list,
            ),
            seq(
                $.expr_list_eos,
                $.cond_eos,
            ),
        ),

        for_stmt: $ => seq(
            $.FOR,
            $.optional_label,
            '(',
            $.for_cond,
            ')',
            $.statement,
        ),

        paren_cond: $ => seq(
            '(',
            $.cond,
            ')',
        ),

        while_stmt: $ => seq(
            $.WHILE,
            $.optional_label,
            $.paren_cond,
            $.statement,
        ),

        do_stmt: $ => choice(
            seq(
                $.DO,
                $.optional_label,
                $.compound_statement,
                $.WHILE,
                '(',
                $.expr,
                ')',
                ';',
            ),
            seq(
                $.DO,
                $.optional_label,
                $.compound_statement,
                ';',
            ),
        ),

        optional_label_target: $ => choice(
            $.CONST_IDENT,
            $.empty,
        ),

        continue_stmt: $ => seq(
            $.CONTINUE,
            $.optional_label_target,
            ';',
        ),

        break_stmt: $ => seq(
            $.BREAK,
            $.optional_label_target,
            ';',
        ),

        nextcase_stmt: $ => choice(
            seq(
                $.NEXTCASE,
                $.CONST_IDENT,
                ':',
                $.expr,
                ';',
            ),
            seq(
                $.NEXTCASE,
                $.expr,
                ';',
            ),
            seq(
                $.NEXTCASE,
                $.CONST_IDENT,
                ':',
                $.type,
                ';',
            ),
            seq(
                $.NEXTCASE,
                $.type,
                ';',
            ),
            seq(
                $.NEXTCASE,
                $.CONST_IDENT,
                ':',
                $.DEFAULT,
                ';',
            ),
            seq(
                $.NEXTCASE,
                $.DEFAULT,
                ';',
            ),
            seq(
                $.NEXTCASE,
                ';',
            ),
        ),

        foreach_var: $ => choice(
            seq(
                $.optional_type,
                '&',
                $.IDENT,
            ),
            seq(
                $.optional_type,
                $.IDENT,
            ),
            seq(
                '&',
                $.IDENT,
            ),
            $.IDENT,
        ),

        foreach_vars: $ => choice(
            $.foreach_var,
            seq(
                $.foreach_var,
                ',',
                $.foreach_var,
            ),
        ),

        foreach_stmt: $ => seq(
            $.FOREACH,
            $.optional_label,
            '(',
            $.foreach_vars,
            ':',
            $.expr,
            ')',
            $.statement,
        ),

        defer_stmt: $ => choice(
            seq(
                $.DEFER,
                $.statement,
            ),
            seq(
                $.DEFER,
                $.TRY,
                $.statement,
            ),
            seq(
                $.DEFER,
                $.CATCH,
                $.statement,
            ),
        ),

        ct_if_stmt: $ => choice(
            seq(
                $.CT_IF,
                $.constant_expr,
                ':',
                $.opt_stmt_list,
                $.CT_ENDIF,
            ),
            seq(
                $.CT_IF,
                $.constant_expr,
                ':',
                $.opt_stmt_list,
                $.CT_ELSE,
                $.opt_stmt_list,
                $.CT_ENDIF,
            ),
        ),

        assert_expr_list: $ => choice(
            $.expr,
            seq(
                $.expr,
                ',',
                $.assert_expr_list,
            ),
        ),

        assert_stmt: $ => choice(
            seq(
                $.ASSERT,
                '(',
                $.expr,
                ')',
                ';',
            ),
            seq(
                $.ASSERT,
                '(',
                $.expr,
                ',',
                $.assert_expr_list,
                ')',
                ';',
            ),
        ),

        asm_stmts: $ => choice(
            $.asm_stmt,
            seq(
                $.asm_stmts,
                $.asm_stmt,
            ),
        ),

        asm_instr: $ => choice(
            $.INT,
            $.IDENT,
            seq(
                $.INT,
                '.',
                $.IDENT,
            ),
            seq(
                $.IDENT,
                '.',
                $.IDENT,
            ),
        ),

        asm_addr: $ => choice(
            $.asm_expr,
            seq(
                $.asm_expr,
                $.additive_op,
                $.asm_expr,
            ),
            seq(
                $.asm_expr,
                $.additive_op,
                $.asm_expr,
                '*',
                $.INTEGER,
            ),
            seq(
                $.asm_expr,
                $.additive_op,
                $.asm_expr,
                '*',
                $.INTEGER,
                $.additive_op,
                $.INTEGER,
            ),
            seq(
                $.asm_expr,
                $.additive_op,
                $.asm_expr,
                $.shift_op,
                $.INTEGER,
            ),
            seq(
                $.asm_expr,
                $.additive_op,
                $.asm_expr,
                $.additive_op,
                $.INTEGER,
            ),
        ),

        asm_expr: $ => choice(
            $.CT_IDENT,
            $.CT_CONST_IDENT,
            $.IDENT,
            seq(
                '&',
                $.IDENT,
            ),
            $.CONST_IDENT,
            $.REAL,
            $.INTEGER,
            seq(
                '(',
                $.expr,
                ')',
            ),
            seq(
                '[',
                $.asm_addr,
                ']',
                $.asm_exprs,
            ),
        ),

        asm_stmt: $ => choice(
            seq(
                $.asm_instr,
                $.asm_exprs,
                ';',
            ),
            seq(
                $.asm_instr,
                ';',
            ),
        ),

        asm_block_stmt: $ => choice(
            seq(
                $.ASM,
                '(',
                $.constant_expr,
                ')',
                ';',
            ),
            seq(
                $.ASM,
                '(',
                $.constant_expr,
                ')',
                $.AT_IDENT,
                ';',
            ),
            seq(
                $.ASM,
                '{',
                $.asm_stmts,
                '}',
            ),
            seq(
                $.ASM,
                $.AT_IDENT,
                '{',
                $.asm_stmts,
                '}',
            ),
            seq(
                $.ASM,
                '{',
                '}',
            ),
            seq(
                $.ASM,
                $.AT_IDENT,
                '{',
                '}',
            ),
        ),

        statement: $ => choice(
            $.compound_statement,
            $.var_stmt,
            $.declaration_stmt,
            $.return_stmt,
            $.if_stmt,
            $.while_stmt,
            $.defer_stmt,
            $.switch_stmt,
            $.do_stmt,
            $.for_stmt,
            $.foreach_stmt,
            $.continue_stmt,
            $.break_stmt,
            $.nextcase_stmt,
            $.asm_block_stmt,
            $.ct_echo_stmt,
            $.ct_assert_stmt,
            $.ct_if_stmt,
            $.ct_switch_stmt,
            $.ct_foreach_stmt,
            $.ct_for_stmt,
            seq(
                $.expr_no_list,
                ';',
            ),
            $.assert_stmt,
            ';',
        ),

        compound_statement: $ => seq(
            '{',
            $.opt_stmt_list,
            '}',
        ),

        statement_list: $ => choice(
            $.statement,
            seq(
                $.statement_list,
                $.statement,
            ),
        ),

        opt_stmt_list: $ => choice(
            $.statement_list,
            $.empty,
        ),

        switch_stmt: $ => choice(
            seq(
                $.SWITCH,
                $.optional_label,
                '{',
                $.switch_body,
                '}',
            ),
            seq(
                $.SWITCH,
                $.optional_label,
                '{',
                '}',
            ),
            seq(
                $.SWITCH,
                $.optional_label,
                $.paren_cond,
                '{',
                $.switch_body,
                '}',
            ),
            seq(
                $.SWITCH,
                $.optional_label,
                $.paren_cond,
                '{',
                '}',
            ),
        ),

        expression_list: $ => choice(
            $.decl_or_expr,
            seq(
                $.expression_list,
                ',',
                $.decl_or_expr,
            ),
        ),

        optional_label: $ => choice(
            seq(
                $.CONST_IDENT,
                ':',
            ),
            $.empty,
        ),

        ct_assert_stmt: $ => choice(
            seq(
                $.CT_ASSERT,
                $.constant_expr,
                ':',
                $.constant_expr,
                ';',
            ),
            seq(
                $.CT_ASSERT,
                $.constant_expr,
                ';',
            ),
            seq(
                $.CT_ERROR,
                $.constant_expr,
                ';',
            ),
        ),

        ct_include_stmt: $ => seq(
            $.CT_INCLUDE,
            $.string_expr,
            ';',
        ),

        ct_echo_stmt: $ => seq(
            $.CT_ECHO,
            $.constant_expr,
            ';',
            $.bitstruct_declaration,
        ),

        bitstruct_defs: $ => choice(
            $.bitstruct_def,
            seq(
                $.bitstruct_defs,
                $.bitstruct_def,
            ),
        ),

        bitstruct_simple_defs: $ => choice(
            seq(
                $.base_type,
                $.IDENT,
                ';',
            ),
            seq(
                $.bitstruct_simple_defs,
                $.base_type,
                $.IDENT,
                ';',
            ),
        ),

        bitstruct_def: $ => choice(
            seq(
                $.base_type,
                $.IDENT,
                ':',
                $.constant_expr,
                $.DOTDOT,
                $.constant_expr,
                ';',
            ),
            seq(
                $.base_type,
                $.IDENT,
                ':',
                $.constant_expr,
                ';',
            ),
        ),

        attribute_name: $ => choice(
            $.AT_IDENT,
            $.AT_TYPE_IDENT,
            seq(
                $.path,
                $.AT_TYPE_IDENT,
            ),
        ),

        attribute_operator_expr: $ => choice(
            seq(
                '&',
                '[',
                ']',
            ),
            seq(
                '[',
                ']',
                '=',
            ),
            seq(
                '[',
                ']',
            ),
        ),

        attr_param: $ => choice(
            $.attribute_operator_expr,
            $.constant_expr,
        ),

        attribute_param_list: $ => choice(
            $.attr_param,
            seq(
                $.attribute_param_list,
                ',',
                $.attr_param,
            ),
        ),

        attribute: $ => choice(
            $.attribute_name,
            seq(
                $.attribute_name,
                '(',
                $.attribute_param_list,
                ')',
            ),
        ),

        attribute_list: $ => choice(
            $.attribute,
            seq(
                $.attribute_list,
                $.attribute,
            ),
        ),

        opt_attributes: $ => choice(
            $.attribute_list,
            $.empty,
        ),

        trailing_block_param: $ => choice(
            $.AT_IDENT,
            seq(
                $.AT_IDENT,
                '(',
                ')',
            ),
            seq(
                $.AT_IDENT,
                '(',
                $.parameters,
                ')',
            ),
        ),

        macro_params: $ => choice(
            $.parameters,
            seq(
                $.parameters,
                ';',
                $.trailing_block_param,
            ),
            seq(
                ';',
                $.trailing_block_param,
            ),
            $.empty,
        ),

        macro_func_body: $ => choice(
            seq(
                $.implies_body,
                ';',
            ),
            $.compound_statement,
        ),

        macro_declaration: $ => seq(
            $.MACRO,
            $.macro_header,
            '(',
            $.macro_params,
            ')',
            $.opt_attributes,
            $.macro_func_body,
        ),

        struct_or_union: $ => choice(
            $.STRUCT,
            $.UNION,
        ),

        struct_declaration: $ => seq(
            $.struct_or_union,
            $.TYPE_IDENT,
            $.opt_interface_impl,
            $.opt_attributes,
            $.struct_body,
        ),

        struct_body: $ => seq(
            '{',
            $.struct_declaration_list,
            '}',
        ),

        struct_declaration_list: $ => choice(
            $.struct_member_decl,
            seq(
                $.struct_declaration_list,
                $.struct_member_decl,
            ),
        ),

        enum_params: $ => choice(
            $.enum_param_decl,
            seq(
                $.enum_params,
                ',',
                $.enum_param_decl,
            ),
        ),

        enum_param_list: $ => choice(
            seq(
                '(',
                $.enum_params,
                ')',
            ),
            seq(
                '(',
                ')',
            ),
            $.empty,
        ),

        struct_member_decl: $ => choice(
            seq(
                $.type,
                $.identifier_list,
                $.opt_attributes,
                ';',
            ),
            seq(
                $.struct_or_union,
                $.IDENT,
                $.opt_attributes,
                $.struct_body,
            ),
            seq(
                $.struct_or_union,
                $.opt_attributes,
                $.struct_body,
            ),
            seq(
                $.BITSTRUCT,
                ':',
                $.type,
                $.opt_attributes,
                $.bitstruct_body,
            ),
            seq(
                $.BITSTRUCT,
                $.IDENT,
                ':',
                $.type,
                $.opt_attributes,
                $.bitstruct_body,
            ),
            seq(
                $.INLINE,
                $.type,
                $.IDENT,
                $.opt_attributes,
                ';',
            ),
            seq(
                $.INLINE,
                $.type,
                $.opt_attributes,
                ';',
            ),
        ),

        enum_spec: $ => choice(
            seq(
                ':',
                $.type,
                $.enum_param_list,
            ),
            $.empty,
        ),

        enum_declaration: $ => seq(
            $.ENUM,
            $.TYPE_IDENT,
            $.opt_interface_impl,
            $.enum_spec,
            $.opt_attributes,
            '{',
            $.enum_list,
            '}',
        ),

        faults: $ => choice(
            $.CONST_IDENT,
            seq(
                $.faults,
                ',',
                $.CONST_IDENT,
            ),
        ),

        fault_declaration: $ => choice(
            seq(
                $.FAULT,
                $.TYPE_IDENT,
                $.opt_interface_impl,
                $.opt_attributes,
                '{',
                $.faults,
                '}',
            ),
            seq(
                $.FAULT,
                $.TYPE_IDENT,
                $.opt_interface_impl,
                $.opt_attributes,
                '{',
                $.faults,
                ',',
                '}',
            ),
        ),

        func_macro_name: $ => choice(
            $.IDENT,
            $.AT_IDENT,
        ),

        func_header: $ => choice(
            seq(
                $.optional_type,
                $.type,
                '.',
                $.func_macro_name,
            ),
            seq(
                $.optional_type,
                $.func_macro_name,
            ),
        ),

        macro_header: $ => choice(
            $.func_header,
            seq(
                $.type,
                '.',
                $.func_macro_name,
            ),
            $.func_macro_name,
        ),

        fn_parameter_list: $ => choice(
            seq(
                '(',
                $.parameters,
                ')',
            ),
            seq(
                '(',
                ')',
            ),
        ),

        parameters: $ => choice(
            seq(
                $.parameter,
                '=',
                $.expr,
            ),
            $.parameter,
            seq(
                $.parameters,
                ',',
                $.parameter,
            ),
            seq(
                $.parameters,
                ',',
                $.parameter,
                '=',
                $.expr,
            ),
        ),

        parameter: $ => choice(
            seq(
                $.type,
                $.IDENT,
                $.opt_attributes,
            ),
            seq(
                $.type,
                $.ELLIPSIS,
                $.IDENT,
                $.opt_attributes,
            ),
            seq(
                $.type,
                $.ELLIPSIS,
                $.CT_IDENT,
            ),
            seq(
                $.type,
                $.CT_IDENT,
            ),
            seq(
                $.type,
                $.ELLIPSIS,
                $.opt_attributes,
            ),
            seq(
                $.type,
                $.HASH_IDENT,
                $.opt_attributes,
            ),
            seq(
                $.type,
                '&',
                $.IDENT,
                $.opt_attributes,
            ),
            seq(
                $.type,
                $.opt_attributes,
            ),
            seq(
                '&',
                $.IDENT,
                $.opt_attributes,
            ),
            seq(
                $.HASH_IDENT,
                $.opt_attributes,
            ),
            $.ELLIPSIS,
            seq(
                $.IDENT,
                $.opt_attributes,
            ),
            seq(
                $.IDENT,
                $.ELLIPSIS,
                $.opt_attributes,
            ),
            $.CT_IDENT,
            seq(
                $.CT_IDENT,
                $.ELLIPSIS,
            ),
        ),

        func_defintion_decl: $ => seq(
            $.FN,
            $.func_header,
            $.fn_parameter_list,
            $.opt_attributes,
            ';',
        ),

        func_definition: $ => choice(
            $.func_defintion_decl,
            seq(
                $.FN,
                $.func_header,
                $.fn_parameter_list,
                $.opt_attributes,
                $.macro_func_body,
            ),
        ),

        const_declaration: $ => choice(
            seq(
                $.CONST,
                $.CONST_IDENT,
                $.opt_attributes,
                '=',
                $.expr,
                ';',
            ),
            seq(
                $.CONST,
                $.type,
                $.CONST_IDENT,
                $.opt_attributes,
                '=',
                $.expr,
                ';',
            ),
        ),

        func_typedef: $ => seq(
            $.FN,
            $.optional_type,
            $.fn_parameter_list,
        ),

        opt_inline: $ => choice(
            $.INLINE,
            $.empty,
        ),

        generic_parameters: $ => choice(
            $.expr,
            $.type,
            seq(
                $.generic_parameters,
                ',',
                $.expr,
            ),
            seq(
                $.generic_parameters,
                ',',
                $.type,
            ),
        ),

        typedef_type: $ => choice(
            $.func_typedef,
            $.type,
        ),

        multi_declaration: $ => choice(
            seq(
                ',',
                $.IDENT,
            ),
            seq(
                $.multi_declaration,
                ',',
                $.IDENT,
            ),
        ),

        global_storage: $ => choice(
            $.TLOCAL,
            $.empty,
        ),

        global_declaration: $ => choice(
            seq(
                $.global_storage,
                $.optional_type,
                $.IDENT,
                $.opt_attributes,
                ';',
            ),
            seq(
                $.global_storage,
                $.optional_type,
                $.IDENT,
                $.multi_declaration,
                $.opt_attributes,
                ';',
            ),
            seq(
                $.global_storage,
                $.optional_type,
                $.IDENT,
                $.opt_attributes,
                '=',
                $.expr,
                ';',
            ),
        ),

        opt_tl_stmts: $ => choice(
            $.top_level_statements,
            $.empty,
        ),

        tl_ct_case: $ => choice(
            seq(
                $.CT_CASE,
                $.constant_expr,
                ':',
                $.opt_tl_stmts,
            ),
            seq(
                $.CT_CASE,
                $.type,
                ':',
                $.opt_tl_stmts,
            ),
            seq(
                $.CT_DEFAULT,
                ':',
                $.opt_tl_stmts,
            ),
        ),

        tl_ct_switch_body: $ => choice(
            $.tl_ct_case,
            seq(
                $.tl_ct_switch_body,
                $.tl_ct_case,
            ),
        ),

        define_attribute: $ => choice(
            seq(
                $.AT_TYPE_IDENT,
                '(',
                $.parameters,
                ')',
                $.opt_attributes,
                '=',
                '{',
                $.opt_attributes,
                '}',
            ),
            seq(
                $.AT_TYPE_IDENT,
                $.opt_attributes,
                '=',
                '{',
                $.opt_attributes,
                '}',
            ),
        ),

        generic_expr: $ => seq(
            $.LGENPAR,
            $.generic_parameters,
            $.RGENPAR,
        ),

        opt_generic_parameters: $ => choice(
            $.generic_expr,
            $.empty,
        ),

        define_ident: $ => choice(
            seq(
                $.IDENT,
                '=',
                $.path_ident,
                $.opt_generic_parameters,
            ),
            seq(
                $.CONST_IDENT,
                '=',
                $.path_const,
                $.opt_generic_parameters,
            ),
            seq(
                $.AT_IDENT,
                '=',
                $.path_at_ident,
                $.opt_generic_parameters,
            ),
        ),

        define_declaration: $ => choice(
            seq(
                $.DEF,
                $.define_ident,
                $.opt_attributes,
                ';',
            ),
            seq(
                $.DEF,
                $.define_attribute,
                $.opt_attributes,
                ';',
            ),
            seq(
                $.DEF,
                $.TYPE_IDENT,
                $.opt_attributes,
                '=',
                $.typedef_type,
                $.opt_attributes,
                ';',
            ),
        ),

        interface_body: $ => choice(
            $.func_defintion_decl,
            seq(
                $.interface_body,
                $.func_defintion_decl,
            ),
        ),

        interface_declaration: $ => choice(
            seq(
                $.INTERFACE,
                $.TYPE_IDENT,
                '{',
                '}',
            ),
            seq(
                $.INTERFACE,
                $.TYPE_IDENT,
                '{',
                $.interface_body,
                '}',
            ),
        ),

        distinct_declaration: $ => seq(
            $.DISTINCT,
            $.TYPE_IDENT,
            $.opt_interface_impl,
            $.opt_attributes,
            '=',
            $.opt_inline,
            $.type,
            ';',
        ),

        tl_ct_if: $ => seq(
            $.CT_IF,
            $.constant_expr,
            ':',
            $.opt_tl_stmts,
            $.tl_ct_if_tail,
        ),

        tl_ct_if_tail: $ => choice(
            $.CT_ENDIF,
            seq(
                $.CT_ELSE,
                $.opt_tl_stmts,
                $.CT_ENDIF,
            ),
        ),

        tl_ct_switch: $ => seq(
            $.ct_switch,
            $.tl_ct_switch_body,
            $.CT_ENDSWITCH,
        ),

        module_param: $ => choice(
            $.CONST_IDENT,
            $.TYPE_IDENT,
        ),

        module_params: $ => choice(
            $.module_param,
            seq(
                $.module_params,
                ',',
                $.module_param,
            ),
        ),

        module: $ => choice(
            seq(
                $.MODULE,
                $.path_ident,
                $.opt_attributes,
                ';',
            ),
            seq(
                $.MODULE,
                $.path_ident,
                $.LGENPAR,
                $.module_params,
                $.RGENPAR,
                $.opt_attributes,
                ';',
            ),
        ),

        import_paths: $ => choice(
            $.path_ident,
            seq(
                $.path_ident,
                ',',
                $.path_ident,
            ),
        ),

        import_decl: $ => seq(
            $.IMPORT,
            $.import_paths,
            $.opt_attributes,
            ';',
        ),

        translation_unit: $ => choice(
            $.top_level_statements,
            $.empty,
        ),

        top_level_statements: $ => choice(
            $.top_level,
            seq(
                $.top_level_statements,
                $.top_level,
            ),
        ),

        opt_extern: $ => choice(
            $.EXTERN,
            $.empty,
        ),

        top_level: $ => choice(
            $.module,
            $.import_decl,
            seq(
                $.opt_extern,
                $.func_definition,
            ),
            seq(
                $.opt_extern,
                $.const_declaration,
            ),
            seq(
                $.opt_extern,
                $.global_declaration,
            ),
            $.ct_assert_stmt,
            $.ct_echo_stmt,
            $.ct_include_stmt,
            $.tl_ct_if,
            $.tl_ct_switch,
            $.struct_declaration,
            $.fault_declaration,
            $.enum_declaration,
            $.macro_declaration,
            $.define_declaration,
            $.bitstruct_declaration,
            $.distinct_declaration,
            $.interface_declaration,
        ),

    }
});