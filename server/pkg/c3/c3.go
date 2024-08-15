package c3

func Keywords() []string {
	return []string{
		"void", "bool", "char", "double",
		"float", "float16", "int128", "ichar",
		"int", "iptr", "isz", "long",
		"short", "uint128", "uint", "ulong",
		"uptr", "ushort", "usz", "float128",
		"any", "anyfault", "typeid", "assert",
		"asm", "bitstruct", "break", "case",
		"catch", "const", "continue", "def",
		"default", "defer", "distinct", "do",
		"else", "enum", "extern", "false",
		"fault", "for", "foreach", "foreach_r",
		"fn", "tlocal", "if", "inline",
		"import", "macro", "module", "nextcase",
		"null", "return", "static", "struct",
		"switch", "true", "try", "union",
		"var", "while",

		"$alignof", "$assert", "$case", "$default",
		"$defined", "$echo", "$embed", "$exec",
		"$else", "$endfor", "$endforeach", "$endif",
		"$endswitch", "$eval", "$evaltype", "$error",
		"$extnameof", "$for", "$foreach", "$if",
		"$include", "$nameof", "$offsetof", "$qnameof",
		"$sizeof", "$stringify", "$switch", "$typefrom",
		"$typeof", "$vacount", "$vatype", "$vaconst",
		"$varef", "$vaarg", "$vaexpr", "$vasplat",
	}
}

func IsLanguageKeyword(symbol string) bool {
	keywords := Keywords()

	for _, w := range keywords {
		if w == symbol {
			return true
		}
	}
	return false
}
