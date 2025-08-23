package model

type AITokenizedWord struct {
	Token          string
	Furigana       string
	Lemma          string
	LemmaFurigana  string
	Conjugation    string
	GrammarID      int
	EnglishMeaning []string
	PartOfSpeechID int
	Position       int
	JLPTLevel      *int
	FrequencyRank  *int
}
