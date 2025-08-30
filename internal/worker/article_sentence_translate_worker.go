package worker

import (
	"encoding/json"
	"nhknewseasybkend/internal/service"
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
	"os"
	"regexp"
)

func ArticleSentenceTranslateWorker(s types.Sentence) (types.ArticleSentenceResult, error) {
	util.Log("[ArticleSentenceTranslateWorker] Translating sentence: '"+s.JP+"'", util.LogTypeLog)
	result, err := TokenizeSentenceWithAI(s.JP)
	if err != nil || !result.Success {
		util.Log("[ArticleSentenceTranslateWorker] Error tokenizing sentence: "+err.Error(), util.LogTypeError)
		return types.ArticleSentenceResult{}, err
	}
	util.Log("[ArticleSentenceTranslateWorker] Tokenization result: ", util.LogTypeLog)
	EnhanceTokenizedWords(&result)
	util.Log("[ArticleSentenceTranslateWorker] Enhancement complete.", util.LogTypeLog)

	// AI translation and explanation step
	translatedSentence, err := TranslateJapaneseToEnglish(s)
	if err != nil {
		util.Log("[ArticleSentenceTranslateWorker] Error translating sentence: "+err.Error(), util.LogTypeError)
		// fallback: leave EN and Eval empty
	}

	return types.ArticleSentenceResult{Sentence: types.Sentence{JP: s.JP, EN: translatedSentence.EN, Eval: translatedSentence.Eval, Words: result.Words}}, nil
}

// EnhanceTokenizedWords calls a cheaper AI model to add meanings, JLPT, and frequency to tokenized words
func EnhanceTokenizedWords(result *types.AITokenizationResult) {
	// Prepare input for enhancement, skipping words already in LocalWordStore
	var newWords []map[string]string
	var skippedIndices []int
	var skippedWords []types.AITokenizedWord
	for i, w := range result.Words {
		if dbWord, exists := service.LocalWordStore[w.Token]; exists {
			w.ID = dbWord.ID
			skippedIndices = append(skippedIndices, i)
			skippedWords = append(skippedWords, w)
			continue
		}
		newWords = append(newWords, map[string]string{
			"lemma": w.Lemma,
			"token": w.Token,
		})
	}

	// Build enhancement prompt
	enhancementInput, _ := json.Marshal(newWords)
	enhancementPrompt := "For each of the following Japanese words, provide enhanced information:\n" +
		"- english_meaning: array of contextual English meanings\n" +
		"- jlpt_level: JLPT level (1-5) for this word, or null if not applicable. Consider: N5=5 (most basic), N4=4, N3=3, N2=2, N1=1 (most advanced)\n" +
		"- frequency_rank: estimated frequency rank for this word (lower numbers = more common), or null if very rare/unknown\n" +
		"- confidence: a float value between 0 and 1 (by 2 decimal places so 90% is .90 etc) (0 being no confidence, 1 being full confidence) representing your confidence in the accuracy of the meanings and metadata for this word\n\n" +
		"Input words: " + string(enhancementInput) + "\n\n" +
		"Return ONLY a valid JSON array with enhanced information for each word in the same order."

	// Call cheaper AI model for enhancement
	modelLow := os.Getenv("AI_MODEL_LOW")
	enhancementMessages := []service.OpenAIChatMessage{
		{Role: "system", Content: "You are a Japanese linguistics expert. Return only valid JSON."},
		{Role: "user", Content: enhancementPrompt},
	}
	enhancementResponse, err := service.CallOpenAIChatCompletion(modelLow, 1, enhancementMessages)
	util.Log("[ArticleSentenceTranslateWorker] OpenAI enhancement raw response: "+enhancementResponse, util.LogTypeLog)
	if err != nil {
		util.Log("[ArticleSentenceTranslateWorker] Error enhancing tokens: "+err.Error(), util.LogTypeError)
		return
	}

	// Parse enhancement response
	var enhanced []struct {
		EnglishMeaning []string `json:"english_meaning"`
		JLPTLevel      *int     `json:"jlpt_level"`
		FrequencyRank  *int     `json:"frequency_rank"`
		Confidence     float64  `json:"confidence"`
	}
	re := regexp.MustCompile("(?s)\x60\x60\x60(?:json)?\\s*(\\[.*?\\])\\s*\x60\x60\x60")
	matches := re.FindStringSubmatch(enhancementResponse)
	var enhancementJson string
	if len(matches) > 1 {
		enhancementJson = matches[1]
	} else {
		enhancementJson = enhancementResponse
	}
	if err := json.Unmarshal([]byte(enhancementJson), &enhanced); err != nil {
		util.Log("[ArticleSentenceTranslateWorker] Error parsing enhancement response: "+err.Error(), util.LogTypeError)
		return
	}

	// Update result.Words with enhanced info, skipping original words
	enhancedIdx := 0
	for i := range result.Words {
		// If this index was skipped, restore the original word
		found := false
		for j, idx := range skippedIndices {
			if i == idx {
				result.Words[i] = skippedWords[j]
				found = true
				break
			}
		}
		if found {
			continue
		}
		if enhancedIdx < len(enhanced) {
			result.Words[i].EnglishMeaning = enhanced[enhancedIdx].EnglishMeaning
			result.Words[i].JLPTLevel = enhanced[enhancedIdx].JLPTLevel
			result.Words[i].FrequencyRank = enhanced[enhancedIdx].FrequencyRank
			result.Words[i].Confidence = enhanced[enhancedIdx].Confidence
			enhancedIdx++
		}
	}
	util.Log("[ArticleSentenceTranslateWorker] Enhancement complete.", util.LogTypeLog)
}

// TokenizeSentenceWithAI sends a sentence to the AI model for tokenization
func TokenizeSentenceWithAI(sentence string) (types.AITokenizationResult, error) {
	model := os.Getenv("AI_MODEL_HIGH") // Use high model for tokenization
	util.Log("[TokenizeSentenceWithAI] Tokenizing sentence with model: "+model, util.LogTypeLog)

	// Prepare prompt for AI including conjugation info
	// Prepare conjugation and grammar info for prompt
	// NOTE: To avoid import cycle, caller must provide conjugationPrompt and grammarStr
	conjugationPrompt := "none, dictionary_form, past, te_form, masu_form, i_adj_past, i_adj_negative, na_adj_past, na_adj_negative" // Example, replace with actual values or pass as param
	grammarStr := "1=は, 2=が, 3=を"                                                                                                    // Example, replace with actual values or pass as param

	// Build the basic tokenization prompt
	basicTokenizationPrompt := "Analyze this Japanese sentence and break it down into meaningful tokens.\n\n" +
		"1. CRITICAL: Keep complete verb forms together as single units.\n" +
		"2. CRITICAL: Particles (に, を, が, は, で, から, の, etc.) MUST be split as their own tokens.\n" +
		"3. DO NOT break down verb conjugations into morphemes.\n" +
		"4. CRITICAL: Treat date/time expressions and counters as single words.\n" +
		"5. For verbal nouns with auxiliary forms (e.g., 参加しながら), split as: '参加' (名詞), 'しながら' (助詞 or 動詞 as appropriate).\n\n" +
		"For each word, provide ONLY basic information:\n" +
		"- token: string as in the sentence\n" +
		"- part_of_speech_id: Japanese POS assign the correct pos id (must be a number) from the following list: \n" +
		"[POS LIST HERE]\n" +
		"- lemma: lemma form of the word for dictionary lookup\n" +
		"- conjugation: CRITICAL: Must be exactly one of these standardized values:\n" + conjugationPrompt + "\n\n" +
		"CONJUGATION RULES:\n" +
		"- Use \"none\" for particles, nouns, and unconjugated words\n" +
		"- Use \"dictionary_form\" for base/plain forms of verbs and adjectives\n" +
		"- Use specific forms like \"past\", \"te_form\", \"masu_form\" for conjugated verbs\n" +
		"- Use \"i_adj_past\", \"i_adj_negative\" etc. for i-adjectives\n" +
		"- Use \"na_adj_past\", \"na_adj_negative\" etc. for na-adjectives\n" +
		"- DO NOT use phrases like \"past tense\" - use exact values like \"past\"\n\n" +
		"- grammar_id: assign the correct grammar id (must be a number) from the following list:\n" + grammarStr + "\n" +
		"- token_furigana: furigana for all kanji in the token, with each kanji's reading in hiragana and enclosed in brackets, and all kana left outside brackets. If the token contains only kana (no kanji), do not use brackets. Examples: 学校 → [がっ][こう], 子ども → [こ]ども, 奈良市 → [な][ら][し], しながら → しながら, する → する\n" +
		"- lemma_furigana: furigana for the lemma, with each kanji's reading in hiragana and enclosed in brackets, and all kana left outside brackets. If the lemma contains only kana (no kanji), do not use brackets. Examples: 学校 → [がっ][こう], 子ども → [こ]ども, 奈良市 → [な][ら][し], しながら → しながら, する → する\n\n" +
		"CRITICAL FURIGANA RULES:\n" +
		"- If word contains only kana (hiragana/katakana, no kanji): NO brackets → Example: しながら → しながら\n" +
		"- If word contains kanji: Each kanji gets brackets around its reading, kana stays outside → Example: 学校 → [がっ][こう]\n" +
		"- Single kanji: [reading] → Example: 音 → [おと]\n" +
		"- Mixed kanji+kana: [kanji_reading]kana → Example: 子ども → [こ]ども\n\n" +
		"Return ONLY a valid JSON array of objects with these fields.\n\n" +
		"Sentence: \"" + sentence + "\""

	// TODO: Insert POS list in prompt if available

	messages := []service.OpenAIChatMessage{
		{Role: "system", Content: "You are a Japanese linguistics expert. Return only valid JSON."},
		{Role: "user", Content: basicTokenizationPrompt},
	}
	aiResponse, err := service.CallOpenAIChatCompletion(model, 1, messages)
	util.Log("[TokenizeSentenceWithAI] OpenAI raw response: "+aiResponse, util.LogTypeLog)
	if err != nil {
		return types.AITokenizationResult{
			Success: false,
			Error:   err.Error(),
			Method:  "ai",
		}, err
	}

	// Try to extract JSON array from response (strip code block if present)
	var jsonStr string
	re := regexp.MustCompile("(?s)\x60\x60\x60(?:json)?\\s*(\\[.*?\\])\\s*\x60\x60\x60")
	matches := re.FindStringSubmatch(aiResponse)
	if len(matches) > 1 {
		jsonStr = matches[1]
	} else {
		jsonStr = aiResponse
	}

	var words []types.AITokenizedWord
	if err := json.Unmarshal([]byte(jsonStr), &words); err != nil {
		return types.AITokenizationResult{
			Success: false,
			Error:   "Failed to parse AI response: " + err.Error(),
			Method:  "ai",
		}, err
	}

	return types.AITokenizationResult{
		Words:   words,
		Success: true,
		Method:  "ai",
	}, nil
}

// TranslateJapaneseToEnglish translates a Japanese sentence to English and provides a grammar explanation for learners
func TranslateJapaneseToEnglish(s types.Sentence) (types.Sentence, error) {
	model := os.Getenv("AI_MODEL_LOW")
	util.Log("[TranslateJapaneseToEnglish] Translating sentence with model: "+model, util.LogTypeLog)

	// Build the translation and explanation prompt (stricter)
	translationPrompt := "Translate the following Japanese sentence to English. Then, provide a simple explanation for a Japanese learner, including grammar points, tricky parts, and how the sentence is structured. If possible, list grammar patterns used.\n\n" +
		"Japanese: \"" + s.JP + "\"\n\n" +
		"IMPORTANT: Your response MUST be a single valid JSON object with exactly two fields: 'translation' (string) and 'explanation' (string).\n" +
		"Do NOT include any extra text, comments, markdown, code blocks, or formatting.\n" +
		"Do NOT wrap the response in triple backticks or any other code block.\n" +
		"Return ONLY the JSON object, nothing else."

	messages := []service.OpenAIChatMessage{
		{Role: "system", Content: "You are a Japanese teacher and translation expert. Return only valid JSON."},
		{Role: "user", Content: translationPrompt},
	}
	aiResponse, err := service.CallOpenAIChatCompletion(model, 1, messages)
	util.Log("[TranslateJapaneseToEnglish] OpenAI raw response: "+aiResponse, util.LogTypeLog)
	if err != nil {
		return s, err
	}

	// Try to extract JSON object from response
	var resp struct {
		Translation string `json:"translation"`
		Explanation string `json:"explanation"`
	}
	re := regexp.MustCompile(`(?s)\{.*?\}`)
	match := re.FindString(aiResponse)
	var jsonStr string
	if match != "" {
		jsonStr = match
	} else {
		jsonStr = aiResponse
	}
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return s, err
	}

	s.EN = resp.Translation
	s.Eval = resp.Explanation
	return s, nil
}
