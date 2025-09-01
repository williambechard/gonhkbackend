package worker

import (
	"encoding/json"
	"fmt"
	"nhknewseasybkend/internal/service"
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
	"os"
	"regexp"
	"time"
	"unicode"
)

// cleanJSONString removes non-printable/control characters from a JSON string
func cleanJSONString(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			out = append(out, r)
		}
	}
	return string(out)
}

// BatchTokenizeSentencesWithAI sends a batch of sentences to the AI model for tokenization
func BatchTokenizeSentencesWithAI(sentences []string) ([]types.AITokenizationResult, error) {
	// True batch API call implementation
	model := os.Getenv("AI_MODEL_HIGH")
	util.Log(fmt.Sprintf("[BatchTokenizeSentencesWithAI] Sending batch of %d sentences", len(sentences)), util.LogTypeLog)
	var results []types.AITokenizationResult
	maxRetries := 3
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Build batch prompt: array of user messages, one per sentence
		var messages []service.OpenAIChatMessage
		for _, s := range sentences {
			// Use same prompt logic as TokenizeSentenceWithAI
			conjugationPrompt := "none, dictionary_form, past, te_form, masu_form, i_adj_past, i_adj_negative, na_adj_past, na_adj_negative"
			grammarStr := "1=は, 2=が, 3=を"
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
				"Sentence: \"" + s + "\""
			messages = append(messages, service.OpenAIChatMessage{Role: "user", Content: basicTokenizationPrompt})
		}
		// Add system message at the start
		messages = append([]service.OpenAIChatMessage{{Role: "system", Content: "You are a Japanese linguistics expert. Return only valid JSON."}}, messages...)
		aiResponses, err := service.CallOpenAIChatCompletion(model, 1, messages)
		if err == nil {
			// Expect aiResponses to be a JSON array of arrays (one per sentence)
			var batchResponses []json.RawMessage
			cleaned := cleanJSONString(aiResponses)
			// Fix: If batch size is 1 and response is array of tokens, wrap in array for batch logic
			if len(sentences) == 1 && len(cleaned) > 0 && cleaned[0] == '[' {
				cleaned = "[" + cleaned + "]"
			}
			if len(cleaned) > 0 && cleaned[0] == '{' {
				cleaned = "[" + cleaned + "]"
			}
			if err := json.Unmarshal([]byte(cleaned), &batchResponses); err != nil {
				util.Log("[BatchTokenizeSentencesWithAI] Error parsing batch response: "+err.Error(), util.LogTypeError)
				continue
			}
			if len(batchResponses) != len(sentences) {
				util.Log(fmt.Sprintf("[BatchTokenizeSentencesWithAI] ERROR: Batch response count (%d) does not match input count (%d). Panicking to abort further processing.", len(batchResponses), len(sentences)), util.LogTypeError)
				panic(fmt.Sprintf("BatchTokenizeSentencesWithAI: batch response count (%d) does not match input count (%d)", len(batchResponses), len(sentences)))
			}
			for _, raw := range batchResponses {
				var words []types.AITokenizedWord
				cleaned := cleanJSONString(string(raw))
				if err := json.Unmarshal([]byte(cleaned), &words); err != nil {
					results = append(results, types.AITokenizationResult{Success: false, Error: "Failed to parse AI response: " + err.Error(), Method: "ai"})
				} else {
					results = append(results, types.AITokenizationResult{Words: words, Success: true, Method: "ai"})
				}
			}
			return results, nil
		}
		backoff := time.Duration(500*(1<<attempt)) * time.Millisecond
		util.Log(fmt.Sprintf("[BatchTokenizeSentencesWithAI] Retry %d for batch due to error: %v (backoff: %v)", attempt+1, err, backoff), util.LogTypeWarn)
		time.Sleep(backoff)
	}
	return results, err
}

// BatchEnhanceTokenizedWords calls AI to enhance a batch of tokenized results
func BatchEnhanceTokenizedWords(results []types.AITokenizationResult) ([]types.AITokenizationResult, error) {
	// True batch API call implementation
	modelLow := os.Getenv("AI_MODEL_LOW")
	util.Log(fmt.Sprintf("[BatchEnhanceTokenizedWords] Sending batch of %d tokenized results", len(results)), util.LogTypeLog)
	maxRetries := 3
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Build batch enhancement prompt: array of user messages, one per tokenized result
		var messages []service.OpenAIChatMessage
		var filteredResults []types.AITokenizationResult
		for _, result := range results {
			var newWords []map[string]string
			for _, w := range result.Words {
				if _, exists := service.LocalWordStore[w.Token]; !exists {
					newWords = append(newWords, map[string]string{
						"lemma": w.Lemma,
						"token": w.Token,
					})
				}
			}
			if len(newWords) == 0 {
				util.Log("[BatchEnhanceTokenizedWords] Skipping enhancement for result with no new words.", util.LogTypeLog)
				continue
			}
			filteredResults = append(filteredResults, result)
			enhancementInput, _ := json.Marshal(newWords)
			grammarListStr := "(No grammar list loaded)"
			if len(service.AllGrammar) > 0 {
				grammarListStr = "Valid grammar_id values (use ONLY these):\n"
				for _, g := range service.AllGrammar {
					grammarListStr += fmt.Sprintf("%d: %s - %s\n", g.ID, g.Name, g.Description)
				}
			}
			enhancementPrompt := "For each of the following Japanese words, provide enhanced information:\n" +
				"- english_meaning: array of contextual English meanings\n" +
				"- jlpt_level: JLPT level (1-5) for this word, or null if not applicable. Consider: N5=5 (most basic), N4=4, N3=3, N2=2, N1=1 (most advanced)\n" +
				"- frequency_rank: estimated frequency rank for this word (lower numbers = more common), or null if very rare/unknown\n" +
				"- confidence: a float value between 0 and 1 (by 2 decimal places so 90% is .90 etc) (0 being no confidence, 1 being full confidence) representing your confidence in the accuracy of the meanings and metadata for this word\n" +
				"- grammar_id: assign the correct grammar_id for each word from the list below, using ONLY valid IDs. If no grammar point applies, use the ID for 'unknown'.\n" +
				"- lemma_furigana: furigana for the lemma, following these rules:\n" +
				"  * If the lemma contains only kana (hiragana/katakana, no kanji): NO brackets. Example: しながら → しながら\n" +
				"  * If the lemma contains kanji: Each kanji gets brackets around its reading, kana stays outside. Example: 学校 → [がっ][こう]\n" +
				"  * Single kanji: [reading]. Example: 音 → [おと]\n" +
				"  * Mixed kanji+kana: [kanji_reading]kana. Example: 子ども → [こ]ども\n" +
				"  * Do not use brackets for kana-only words.\n" +
				grammarListStr + "\n" +
				"Input words: " + string(enhancementInput) + "\n\n" +
				"Return ONLY a valid JSON array with enhanced information for each word in the same order."
			messages = append(messages, service.OpenAIChatMessage{Role: "user", Content: enhancementPrompt})
		}
		// Add system message at the start
		if len(messages) == 0 {
			util.Log("[BatchEnhanceTokenizedWords] No enhancement messages to send (all results had no new words).", util.LogTypeLog)
			return results, nil
		}
		messages = append([]service.OpenAIChatMessage{{Role: "system", Content: "You are a Japanese linguistics expert. Return only valid JSON."}}, messages...)
		enhancementResponses, err := service.CallOpenAIChatCompletion(modelLow, 1, messages)
		if err == nil {
			// Expect enhancementResponses to be a JSON array of arrays (one per tokenized result)
			var batchResponses []json.RawMessage
			cleaned := cleanJSONString(enhancementResponses)
			if len(cleaned) > 0 && cleaned[0] == '{' {
				cleaned = "[" + cleaned + "]"
			}
			if err := json.Unmarshal([]byte(cleaned), &batchResponses); err != nil {
				util.Log("[BatchEnhanceTokenizedWords] Error parsing batch response: "+err.Error(), util.LogTypeError)
				continue
			}
			// Check for batch response count mismatch
			if len(batchResponses) != len(filteredResults) {
				util.Log(fmt.Sprintf("[BatchEnhanceTokenizedWords] ERROR: Batch response count (%d) does not match input count (%d). Panicking to abort further processing.", len(batchResponses), len(filteredResults)), util.LogTypeError)
				panic(fmt.Sprintf("BatchEnhanceTokenizedWords: batch response count (%d) does not match input count (%d)", len(batchResponses), len(filteredResults)))
			}
			// Robustly handle single-item batch responses (object or array)
			if len(filteredResults) == 1 && len(batchResponses) > 1 {
				// Single input, but multiple enhancements returned as separate batchResponses
				var enhanced []struct {
					EnglishMeaning []string `json:"english_meaning"`
					JLPTLevel      *int     `json:"jlpt_level"`
					FrequencyRank  *int     `json:"frequency_rank"`
					Confidence     float64  `json:"confidence"`
					GrammarID      *int     `json:"grammar_id"`
					LemmaFurigana  *string  `json:"lemma_furigana"`
				}
				// Merge all batchResponses into one array
				for _, raw := range batchResponses {
					cleaned := cleanJSONString(string(raw))
					if len(cleaned) > 0 && cleaned[0] == '{' {
						cleaned = "[" + cleaned + "]"
					}
					var single []struct {
						EnglishMeaning []string `json:"english_meaning"`
						JLPTLevel      *int     `json:"jlpt_level"`
						FrequencyRank  *int     `json:"frequency_rank"`
						Confidence     float64  `json:"confidence"`
						GrammarID      *int     `json:"grammar_id"`
						LemmaFurigana  *string  `json:"lemma_furigana"`
					}
					if err := json.Unmarshal([]byte(cleaned), &single); err == nil {
						enhanced = append(enhanced, single...)
					}
				}
				for j := range filteredResults[0].Words {
					if j < len(enhanced) {
						filteredResults[0].Words[j].EnglishMeaning = enhanced[j].EnglishMeaning
						filteredResults[0].Words[j].JLPTLevel = enhanced[j].JLPTLevel
						filteredResults[0].Words[j].FrequencyRank = enhanced[j].FrequencyRank
						filteredResults[0].Words[j].Confidence = enhanced[j].Confidence
						if enhanced[j].GrammarID != nil {
							filteredResults[0].Words[j].GrammarID = *enhanced[j].GrammarID
						}
						if enhanced[j].LemmaFurigana != nil {
							filteredResults[0].Words[j].LemmaFurigana = *enhanced[j].LemmaFurigana
						}
					}
				}
			} else {
				for i, raw := range batchResponses {
					if i >= len(filteredResults) {
						util.Log(fmt.Sprintf("[BatchEnhanceTokenizedWords] WARNING: batchResponses index %d out of bounds for filteredResults (len=%d)", i, len(filteredResults)), util.LogTypeWarn)
						continue
					}
					var enhanced []struct {
						EnglishMeaning []string `json:"english_meaning"`
						JLPTLevel      *int     `json:"jlpt_level"`
						FrequencyRank  *int     `json:"frequency_rank"`
						Confidence     float64  `json:"confidence"`
						GrammarID      *int     `json:"grammar_id"`
						LemmaFurigana  *string  `json:"lemma_furigana"`
					}
					cleaned := cleanJSONString(string(raw))
					if len(cleaned) > 0 && cleaned[0] == '{' {
						cleaned = "[" + cleaned + "]"
					}
					if err := json.Unmarshal([]byte(cleaned), &enhanced); err != nil {
						util.Log("[BatchEnhanceTokenizedWords] Error parsing enhancement response: "+err.Error(), util.LogTypeError)
						continue
					}
					enhancedIdx := 0
					for j := range filteredResults[i].Words {
						if enhancedIdx < len(enhanced) {
							filteredResults[i].Words[j].EnglishMeaning = enhanced[enhancedIdx].EnglishMeaning
							filteredResults[i].Words[j].JLPTLevel = enhanced[enhancedIdx].JLPTLevel
							filteredResults[i].Words[j].FrequencyRank = enhanced[enhancedIdx].FrequencyRank
							filteredResults[i].Words[j].Confidence = enhanced[enhancedIdx].Confidence
							if enhanced[enhancedIdx].GrammarID != nil {
								filteredResults[i].Words[j].GrammarID = *enhanced[enhancedIdx].GrammarID
							}
							if enhanced[enhancedIdx].LemmaFurigana != nil {
								filteredResults[i].Words[j].LemmaFurigana = *enhanced[enhancedIdx].LemmaFurigana
							}
							enhancedIdx++
						}
					}
				}
			}
			// Merge enhanced filteredResults back into results
			idx := 0
			for i := range results {
				if idx < len(filteredResults) && len(filteredResults[idx].Words) == len(results[i].Words) {
					results[i] = filteredResults[idx]
					idx++
				}
			}
			return results, nil
		}
		backoff := time.Duration(500*(1<<attempt)) * time.Millisecond
		util.Log(fmt.Sprintf("[BatchEnhanceTokenizedWords] Retry %d for batch due to error: %v (backoff: %v)", attempt+1, err, backoff), util.LogTypeWarn)
		time.Sleep(backoff)
	}
	return results, err
}

// BatchTranslateJapaneseToEnglish translates a batch of sentences to English
func BatchTranslateJapaneseToEnglish(sentences []types.Sentence) ([]types.Sentence, error) {
	// True batch API call implementation
	model := os.Getenv("AI_MODEL_LOW")
	util.Log(fmt.Sprintf("[BatchTranslateJapaneseToEnglish] Sending batch of %d sentences", len(sentences)), util.LogTypeLog)
	var results []types.Sentence
	maxRetries := 3
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Build batch prompt: array of user messages, one per sentence
		var messages []service.OpenAIChatMessage
		for _, s := range sentences {
			translationPrompt := "Translate the following Japanese sentence to English. Then, provide a simple explanation for a Japanese learner, including grammar points, tricky parts, and how the sentence is structured. If possible, list grammar patterns used.\n\n" +
				"Japanese: \"" + s.JP + "\"\n\n" +
				"IMPORTANT: Your response MUST be a single valid JSON object with exactly two fields: 'translation' (string) and 'explanation' (string).\n" +
				"Do NOT include any extra text, comments, markdown, code blocks, or formatting.\n" +
				"Do NOT wrap the response in triple backticks or any other code block.\n" +
				"Return ONLY the JSON object, nothing else."
			messages = append(messages, service.OpenAIChatMessage{Role: "user", Content: translationPrompt})
		}
		// Add system message at the start
		messages = append([]service.OpenAIChatMessage{{Role: "system", Content: "You are a Japanese teacher and translation expert. Return only valid JSON."}}, messages...)
		aiResponses, err := service.CallOpenAIChatCompletion(model, 1, messages)
		if err == nil {
			// Expect aiResponses to be a JSON array of objects (one per sentence)
			var batchResponses []json.RawMessage
			cleaned := cleanJSONString(aiResponses)
			if len(cleaned) > 0 && cleaned[0] == '{' {
				cleaned = "[" + cleaned + "]"
			}
			if err := json.Unmarshal([]byte(cleaned), &batchResponses); err != nil {
				util.Log("[BatchTranslateJapaneseToEnglish] Error parsing batch response: "+err.Error(), util.LogTypeError)
				continue
			}
			// Refactored: process each sentence individually with retries
			maxRetries := 3
			for i, s := range sentences {
				var translated types.Sentence
				var lastErr error
				for attempt := 0; attempt < maxRetries; attempt++ {
					translated, lastErr = TranslateJapaneseToEnglish(s)
					if lastErr == nil && translated.EN != "" && translated.Eval != "" {
						break
					}
					util.Log(fmt.Sprintf("[BatchTranslateJapaneseToEnglish] Retry %d for sentence %d due to error or missing output: %v", attempt+1, i, lastErr), util.LogTypeWarn)
					time.Sleep(time.Duration(500*(1<<attempt)) * time.Millisecond)
				}
				if translated.EN == "" {
					util.Log(fmt.Sprintf("[BatchTranslateJapaneseToEnglish] MISSING translation for sentence %d: '%s'", i, s.JP), util.LogTypeWarn)
					translated.EN = "[MISSING TRANSLATION]"
				}
				if translated.Eval == "" {
					util.Log(fmt.Sprintf("[BatchTranslateJapaneseToEnglish] MISSING eval for sentence %d: '%s'", i, s.JP), util.LogTypeWarn)
					translated.Eval = "[MISSING EVAL]"
				}
				translated.JP = s.JP
				translated.Words = s.Words
				results = append(results, translated)
			}
			return results, nil
		}
		backoff := time.Duration(500*(1<<attempt)) * time.Millisecond
		util.Log(fmt.Sprintf("[BatchTranslateJapaneseToEnglish] Retry %d for batch due to error: %v (backoff: %v)", attempt+1, err, backoff), util.LogTypeWarn)
		time.Sleep(backoff)
	}
	return results, err
}

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
	// Guard/log if EN or Eval is blank
	if translatedSentence.EN == "" || translatedSentence.Eval == "" {
		util.Log("[MISSING_TRANSLATION_EVAL] JP: '"+s.JP+"' | EN: '"+translatedSentence.EN+"' | Eval: '"+translatedSentence.Eval+"' | Error: "+func() string {
			if err != nil {
				return err.Error()
			} else {
				return ""
			}
		}(), util.LogTypeWarn)
	}

	return types.ArticleSentenceResult{Sentence: types.Sentence{JP: s.JP, EN: translatedSentence.EN, Eval: translatedSentence.Eval, Words: result.Words}}, nil
}

// EnhanceTokenizedWords calls a cheaper AI model to add meanings, JLPT, and frequency to tokenized words
func EnhanceTokenizedWords(result *types.AITokenizationResult) {
	// Prepare input for enhancement, skipping words already in LocalWordStore
	var newWords []map[string]string
	var skippedIndices []int
	var skippedWords []types.AITokenizedWord

	util.Log(fmt.Sprintf("[EnhanceTokenizedWords] Input has %d words, %d skipped, %d new", len(result.Words), len(skippedIndices), len(newWords)), util.LogTypeLog)
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
	// Build grammar list string from service.AllGrammar
	var grammarListStr string
	if len(service.AllGrammar) > 0 {
		util.Log(fmt.Sprintf("[EnhanceTokenizedWords] Grammar list has %d entries", len(service.AllGrammar)), util.LogTypeLog)
		grammarListStr = "Valid grammar_id values (use ONLY these):\n"
		for _, g := range service.AllGrammar {
			grammarListStr += fmt.Sprintf("%d: %s - %s\n", g.ID, g.Name, g.Description)
		}
	} else {
		grammarListStr = "(No grammar list loaded)"
	}

	enhancementPrompt := "For each of the following Japanese words, provide enhanced information:\n" +
		"- english_meaning: array of contextual English meanings\n" +
		"- jlpt_level: JLPT level (1-5) for this word, or null if not applicable. Consider: N5=5 (most basic), N4=4, N3=3, N2=2, N1=1 (most advanced)\n" +
		"- frequency_rank: estimated frequency rank for this word (lower numbers = more common), or null if very rare/unknown\n" +
		"- confidence: a float value between 0 and 1 (by 2 decimal places so 90% is .90 etc) (0 being no confidence, 1 being full confidence) representing your confidence in the accuracy of the meanings and metadata for this word\n" +
		"- grammar_id: assign the correct grammar_id for each word from the list below, using ONLY valid IDs. If no grammar point applies, use the ID for 'unknown'.\n" +
		"- lemma_furigana: furigana for the lemma, following these rules:\n" +
		"  * If the lemma contains only kana (hiragana/katakana, no kanji): NO brackets. Example: しながら → しながら\n" +
		"  * If the lemma contains kanji: Each kanji gets brackets around its reading, kana stays outside. Example: 学校 → [がっ][こう]\n" +
		"  * Single kanji: [reading]. Example: 音 → [おと]\n" +
		"  * Mixed kanji+kana: [kanji_reading]kana. Example: 子ども → [こ]ども\n" +
		"  * Do not use brackets for kana-only words.\n" +
		grammarListStr + "\n" +
		"Input words: " + string(enhancementInput) + "\n\n" +
		"Return ONLY a valid JSON array with enhanced information for each word in the same order."

	// Call cheaper AI model for enhancement
	modelLow := os.Getenv("AI_MODEL_LOW")
	enhancementMessages := []service.OpenAIChatMessage{
		{Role: "system", Content: "You are a Japanese linguistics expert. Return only valid JSON."},
		{Role: "user", Content: enhancementPrompt},
	}
	enhancementResponse, err := service.CallOpenAIChatCompletion(modelLow, 1, enhancementMessages)
	util.Log(fmt.Sprintf("[ArticleSentenceTranslateWorker] OpenAI enhancement response length: %d", len(enhancementResponse)), util.LogTypeLog)
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
		GrammarID      *int     `json:"grammar_id"`
		LemmaFurigana  *string  `json:"lemma_furigana"`
	}
	re := regexp.MustCompile("(?s)\x60\x60\x60(?:json)?\\s*(\\[.*?\\])\\s*\x60\x60\x60")
	matches := re.FindStringSubmatch(enhancementResponse)
	var enhancementJson string
	if len(matches) > 1 {
		enhancementJson = matches[1]
	} else {
		enhancementJson = enhancementResponse
	}
	enhancementJson = cleanJSONString(enhancementJson)
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
			if enhanced[enhancedIdx].GrammarID != nil {
				result.Words[i].GrammarID = *enhanced[enhancedIdx].GrammarID
			}
			if enhanced[enhancedIdx].LemmaFurigana != nil {
				result.Words[i].LemmaFurigana = *enhanced[enhancedIdx].LemmaFurigana
			}
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
	jsonStr = cleanJSONString(jsonStr)
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
	jsonStr = cleanJSONString(jsonStr)
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return s, err
	}

	s.EN = resp.Translation
	s.Eval = resp.Explanation
	return s, nil
}
