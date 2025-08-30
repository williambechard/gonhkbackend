package types

type DBWord struct {
	ID             string   `json:"id"`
	Lemma          string   `json:"lemma"`
	LemmaFurigana  string   `json:"lemma_furigana"`
	Confidence     float64  `json:"confidence"`
	Ai             bool     `json:"ai"`
	Meanings       []string `json:"meanings"`
	PartOfSpeechID int      `json:"part_of_speech_id"`
	Position       int      `json:"position"`
	JLPTLevel      *int     `json:"jlpt_level"`
	FrequencyRank  *int     `json:"frequency_rank"`
}

type AITokenizedWord struct {
	ID             string
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
	Confidence     float64
}

// ArticleSentenceResult groups a sentence with its tokenized words
type ArticleSentenceResult struct {
	Sentence Sentence
	Words    []AITokenizedWord
}

type AITokenizationResult struct {
	Words   []AITokenizedWord
	Success bool
	Error   string
	Method  string
}

// Sentence represents a sentence and its tokenized words
type Sentence struct {
	ID    string
	JP    string
	EN    string
	Eval  string
	Words []AITokenizedWord
}

type ContentBlock struct {
	Title []Sentence
	Img   string
	Body  []Sentence
}

type ArticleContent struct {
	Title   []Sentence
	Video   string
	Date    string
	Summary []Sentence
	Content []ContentBlock
}

type Article struct {
	ID          string
	SourceURL   string
	CategoryID  int
	CreatedAt   string
	ProcessedAt string
	Video       string
	Date        string
}
