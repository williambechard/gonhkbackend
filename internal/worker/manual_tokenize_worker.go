package worker

import (
	"fmt"
	"nhknewseasybkend/internal/dict/kanjidic"
	"nhknewseasybkend/internal/types"
	"nhknewseasybkend/internal/util"
	"os/exec"
	"strings"
)

// Tokenizes a Japanese sentence using MeCab+Unidic and assigns per-kanji furigana using greedy alignment and Kanjidic2 fallback
func MeCabTokenizeWorker(sentence string) ([]types.AITokenizedWord, error) {
	util.Log(fmt.Sprintf("[MeCabTokenizeWorker] Tokenizing: %s", sentence), util.LogTypeLog)
	mecabDictPath := "internal/dict/unidic"
	cmd := exec.Command("mecab", "-d", mecabDictPath)
	cmd.Stdin = strings.NewReader(sentence)
	out, err := cmd.Output()
	if err != nil {
		util.Log(fmt.Sprintf("[MeCabTokenizeWorker] ERROR: mecab CLI failed: %v", err), util.LogTypeError)
		return nil, fmt.Errorf("mecab CLI failed: %w", err)
	}
	util.Log("[MeCabTokenizeWorker] Raw MeCab output:", util.LogTypeLog)
	util.Log(string(out), util.LogTypeLog)

	var kanjiDict map[string]kanjidic.KanjiReadings
	var dictErr error
	if globalKanjidic2 == nil {
		kanjiDict, dictErr = kanjidic.ParseKanjidic2("internal/dict/kanjidic/kanjidic2.xml")
		if dictErr != nil {
			util.Log(fmt.Sprintf("[MeCabTokenizeWorker] ERROR loading Kanjidic2: %v", dictErr), util.LogTypeError)
		} else {
			globalKanjidic2 = kanjiDict
		}
	}
	kanjiDict = globalKanjidic2

	var words []types.AITokenizedWord
	lines := strings.Split(string(out), "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "EOS" || line == "" {
			continue
		}
		util.Log(fmt.Sprintf("[MeCabTokenizeWorker] Processing line %d: %s", i, line), util.LogTypeLog)
		fields := strings.Split(line, "\t")
		surface := fields[0]
		var features []string
		if len(fields) > 1 {
			features = strings.Split(fields[1], ",")
		}
		util.Log(fmt.Sprintf("[MeCabTokenizeWorker] Surface: %s, Features: %v", surface, features), util.LogTypeLog)
		lemma := ""
		if len(features) > 11 {
			lemma = features[11]
		}
		conjugation := ""
		if len(features) > 4 {
			conjugation = features[4]
		}
		furiganaBracketed := ""
		kanjiRunes := []rune(surface)
		for _, r := range kanjiRunes {
			if isKanji(r) && kanjiDict != nil {
				on, kun := kanjidic.GetKanjiReadings(string(r), kanjiDict)
				var dictReading string
				if len(kanjiRunes) > 1 && len(on) > 0 {
					dictReading = on[0]
				} else if len(kun) > 0 {
					dictReading = kun[0]
				} else if len(on) > 0 {
					dictReading = on[0]
				} else {
					dictReading = "?"
				}
				furiganaBracketed += "[" + katakanaToHiragana(dictReading) + "]"
			}
		}
		if furiganaBracketed == "" {
			furiganaBracketed = surface // fallback for kana/non-kanji tokens
		}
		lemmaFurigana := furiganaBracketed
		// If token is all kana, keep original kana and convert katakana to hiragana for LemmaFurigana
		if isKana(surface) {
			lemmaFurigana = katakanaToHiragana(surface)
			lemma = katakanaToHiragana(surface)
		} else if containsKanji(surface) {
			// Convert bracketed readings to hiragana
			lemmaFurigana = convertBracketedKatakanaToHiragana(furiganaBracketed)
			if lemma != "" && isKatakana(lemma) {
				lemma = katakanaToHiragana(lemma)
			}
		} else {
			lemmaFurigana = katakanaToHiragana(lemma)
		}
		word := types.AITokenizedWord{
			Token:         surface,
			Lemma:         lemma,
			Conjugation:   conjugation,
			Furigana:      furiganaBracketed,
			LemmaFurigana: lemmaFurigana,
		}
		util.Log(fmt.Sprintf("[MeCabTokenizeWorker] Token: %s, Lemma: %s, Features: %v, Furigana: %s", word.Token, word.Lemma, features, word.Furigana), util.LogTypeLog)
		words = append(words, word)
	}
	return words, nil
}

// Greedy alignment splits the reading into segments for each kanji
func greedyFuriganaAlignment(surface, reading string, kanjiIndices []int) []string {
	// ...existing code...
	readingRunes := []rune(reading)
	segments := make([]string, 0, len(kanjiIndices))
	if len(kanjiIndices) == 0 {
		return segments
	}
	if len(kanjiIndices) == 2 {
		for split := 1; split < len(readingRunes); split++ {
			seg1 := string(readingRunes[:split])
			seg2 := string(readingRunes[split:])
			if len(seg1) > 0 && len(seg2) > 0 {
				segments = append(segments, seg1, seg2)
				return segments
			}
		}
	} else {
		segLen := len(readingRunes) / len(kanjiIndices)
		remainder := len(readingRunes) % len(kanjiIndices)
		idx := 0
		for i := 0; i < len(kanjiIndices); i++ {
			extra := 0
			if remainder > 0 {
				extra = 1
				remainder--
			}
			end := idx + segLen + extra
			if end > len(readingRunes) {
				end = len(readingRunes)
			}
			segments = append(segments, string(readingRunes[idx:end]))
			idx = end
		}
	}
	return segments
}

func kanjiFuriganaBrackets(word, reading string) string {
	// Defensive: If reading or word is empty, return word
	if len(word) == 0 || len(reading) == 0 {
		return word
	}
	// If word is all kana, just return word
	if isKana(word) {
		return word
	}
	kanjiRunes := []rune(word)
	furiRunes := []rune(reading)
	// Find kanji positions
	var kanjiIndices []int
	for i, r := range kanjiRunes {
		if isKanji(r) {
			kanjiIndices = append(kanjiIndices, i)
		}
	}
	if len(kanjiIndices) == 0 {
		return word
	}
	var segments []string
	if len(kanjiIndices) == 2 && len(furiRunes) > 2 {
		// Assign first kana to first kanji, rest to second
		segments = []string{string(furiRunes[0]), string(furiRunes[1:])}
	} else {
		segments = splitFuriganaSegments(string(furiRunes), len(kanjiIndices))
	}
	var result string
	segmentIdx := 0
	for _, r := range kanjiRunes {
		if isKanji(r) && segmentIdx < len(segments) {
			result += string(r) + "[" + segments[segmentIdx] + "]"
			segmentIdx++
		} else {
			result += string(r)
		}
	}
	return result
}

func isKana(s string) bool {
	for _, r := range s {
		if !(r >= 0x3040 && r <= 0x309F) && !(r >= 0x30A0 && r <= 0x30FF) {
			return false
		}
	}
	return true
}

func isKanji(r rune) bool {
	return r >= 0x4E00 && r <= 0x9FFF
}

func splitFuriganaSegments(furi string, n int) []string {
	runes := []rune(furi)
	segments := make([]string, 0, n)
	segLen := len(runes) / n
	remainder := len(runes) % n
	idx := 0
	for i := 0; i < n; i++ {
		extra := 0
		if remainder > 0 {
			extra = 1
			remainder--
		}
		end := idx + segLen + extra
		if end > len(runes) {
			end = len(runes)
		}
		segments = append(segments, string(runes[idx:end]))
		idx = end
	}
	return segments
}

// globalKanjidic2 caches the loaded dictionary for repeated calls
var globalKanjidic2 map[string]kanjidic.KanjiReadings

// Converts katakana to hiragana
func katakanaToHiragana(s string) string {
	var result []rune
	for _, r := range s {
		// Katakana range: 0x30A1-0x30F6
		if r >= 0x30A1 && r <= 0x30F6 {
			r -= 0x60
		}
		result = append(result, r)
	}
	return string(result)
}

func getField(fields []string, idx int) string {
	if idx < len(fields) {
		return fields[idx]
	}
	return ""
}

// Helper: checks if string contains any kanji
func containsKanji(s string) bool {
	for _, r := range s {
		if isKanji(r) {
			return true
		}
	}
	return false
}

// Helper: checks if string is all katakana
func isKatakana(s string) bool {
	for _, r := range s {
		if !(r >= 0x30A0 && r <= 0x30FF) {
			return false
		}
	}
	return true
}

// Helper: converts bracketed katakana readings to hiragana
func convertBracketedKatakanaToHiragana(s string) string {
	var out strings.Builder
	inBracket := false
	for _, r := range s {
		if r == '[' {
			inBracket = true
			out.WriteRune(r)
			continue
		}
		if r == ']' {
			inBracket = false
			out.WriteRune(r)
			continue
		}
		if inBracket && r >= 0x30A0 && r <= 0x30FF {
			out.WriteRune(r - 0x60)
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}
