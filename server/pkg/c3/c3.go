package c3

// ref: https://c3-lang.org/implementation-details/grammar/#keywords
var keywords = map[string]struct{}{
	"void": {}, "bool": {}, "char": {}, "double": {},
	"float": {}, "float16": {}, "int128": {}, "ichar": {},
	"int": {}, "iptr": {}, "isz": {}, "long": {},
	"short": {}, "uint128": {}, "uint": {}, "ulong": {},
	"uptr": {}, "ushort": {}, "usz": {}, "float128": {},
	"any": {}, "fault": {}, "typeid": {}, "assert": {},
	"asm": {}, "bitstruct": {}, "break": {}, "case": {},
	"catch": {}, "const": {}, "continue": {}, "alias": {},
	"bfloat": {}, "cenum": {}, "faultdef": {}, "interface": {}, "lengthof": {},
	"default": {}, "defer": {}, "typedef": {}, "do": {},
	"else": {}, "enum": {}, "extern": {}, "false": {},
	"for": {}, "foreach": {}, "foreach_r": {}, "fn": {},
	"tlocal": {}, "if": {}, "inline": {}, "import": {},
	"macro": {}, "module": {}, "nextcase": {}, "null": {},
	"return": {}, "static": {}, "struct": {}, "switch": {},
	"true": {}, "try": {}, "union": {}, "var": {},
	"while": {}, "attrdef": {},

	"$alignof": {}, "$assert": {}, "$case": {}, "$default": {},
	"$assignable": {}, "$defined": {}, "$echo": {}, "$embed": {}, "$exec": {},
	"$else": {}, "$endfor": {}, "$endforeach": {}, "$endif": {},
	"$endswitch": {}, "$eval": {}, "$evaltype": {}, "$error": {}, "$feature": {},
	"$extnameof": {}, "$for": {}, "$foreach": {}, "$if": {},
	"$is_const": {}, "$kindof": {},
	"$include": {}, "$nameof": {}, "$offsetof": {}, "$qnameof": {},
	"$sizeof": {}, "$stringify": {}, "$switch": {}, "$typefrom": {},
	"$typeof": {}, "$vacount": {}, "$vatype": {}, "$vaconst": {},
	"$vaarg": {}, "$vaexpr": {}, "$vasplat": {},
}

func Keywords() map[string]struct{} {
	return keywords
}

func IsLanguageKeyword(symbol string) bool {
	_, exists := keywords[symbol]
	return exists
}
