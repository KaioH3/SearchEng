package engine

// stopwordsEN contains common English stopwords.
var stopwordsEN = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "by": true, "do": true, "for": true, "from": true,
	"has": true, "have": true, "he": true, "her": true, "his": true, "how": true,
	"if": true, "in": true, "is": true, "it": true, "its": true,
	"me": true, "my": true, "no": true, "not": true, "of": true, "on": true,
	"or": true, "our": true, "she": true, "so": true,
	"that": true, "the": true, "their": true, "them": true, "then": true,
	"there": true, "these": true, "they": true, "this": true, "to": true,
	"us": true, "was": true, "we": true, "were": true, "what": true,
	"when": true, "where": true, "which": true, "who": true, "why": true,
	"will": true, "with": true, "would": true, "you": true, "your": true,
	"can": true, "could": true, "did": true, "does": true, "had": true,
	"may": true, "might": true, "must": true, "shall": true, "should": true,
	"about": true, "after": true, "all": true, "also": true, "any": true,
	"been": true, "before": true, "being": true, "between": true, "both": true,
	"but": true, "each": true, "just": true, "more": true, "most": true,
	"much": true, "only": true, "other": true, "over": true, "own": true,
	"same": true, "some": true, "such": true, "than": true, "too": true,
	"very": true, "into": true, "through": true, "during": true, "up": true,
	"down": true, "out": true, "off": true,
}

// stopwordsPT contains common Portuguese stopwords.
var stopwordsPT = map[string]bool{
	"de": true, "do": true, "da": true, "dos": true, "das": true,
	"em": true, "no": true, "na": true, "nos": true, "nas": true,
	"um": true, "uma": true, "uns": true, "umas": true,
	"por": true, "para": true, "com": true, "sem": true,
	"que": true, "como": true, "mais": true, "mas": true,
	"se": true, "ou": true, "ao": true, "aos": true,
	"ele": true, "ela": true, "eles": true, "elas": true,
	"seu": true, "sua": true, "seus": true, "suas": true,
	"esse": true, "essa": true, "este": true, "esta": true,
	"isso": true, "isto": true, "aquilo": true,
	"foi": true, "ser": true, "ter": true, "estar": true,
	"muito": true, "entre": true, "sobre": true,
	"quando": true, "onde": true, "qual": true, "quem": true,
	"porque": true, "pela": true, "pelo": true, "pelas": true, "pelos": true,
	"ainda": true, "mesmo": true, "depois": true, "antes": true,
	"desde": true, "cada": true, "outro": true, "outra": true,
	"outros": true, "outras": true, "todo": true, "toda": true,
	"todos": true, "todas": true, "nao": true,
}

// isStopword returns true if the word is a stopword in English or Portuguese.
func isStopword(word string) bool {
	return stopwordsEN[word] || stopwordsPT[word]
}
