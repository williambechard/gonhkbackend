package service

// Main conjugation types - standardized values
var ConjugationTypes = map[string]string{
	"NONE":                 "none",
	"DICTIONARY_FORM":      "dictionary_form",
	"NEGATIVE":             "negative",
	"PAST":                 "past",
	"PAST_NEGATIVE":        "past_negative",
	"TE_FORM":              "te_form",
	"TA_FORM":              "ta_form",
	"MASU_FORM":            "masu_form",
	"MASU_NEGATIVE":        "masu_negative",
	"MASU_PAST":            "masu_past",
	"MASU_PAST_NEGATIVE":   "masu_past_negative",
	"IMPERATIVE":           "imperative",
	"VOLITIONAL":           "volitional",
	"CONDITIONAL":          "conditional",
	"POTENTIAL":            "potential",
	"PASSIVE":              "passive",
	"CAUSATIVE":            "causative",
	"CAUSATIVE_PASSIVE":    "causative_passive",
	"PROGRESSIVE":          "progressive",
	"PROGRESSIVE_PAST":     "progressive_past",
	"NAGARA_FORM":          "nagara_form",
	"BA_FORM":              "ba_form",
	"TARA_FORM":            "tara_form",
	"NAIDE_FORM":           "naide_form",
	"I_ADJ_NEGATIVE":       "i_adj_negative",
	"I_ADJ_PAST":           "i_adj_past",
	"I_ADJ_PAST_NEGATIVE":  "i_adj_past_negative",
	"I_ADJ_ADVERB":         "i_adj_adverb",
	"NA_ADJ_NEGATIVE":      "na_adj_negative",
	"NA_ADJ_PAST":          "na_adj_past",
	"NA_ADJ_PAST_NEGATIVE": "na_adj_past_negative",
	"HONORIFIC":            "honorific",
	"HUMBLE":               "humble",
	"AUXILIARY":            "auxiliary",
	"PARTICLE":             "particle",
}

// Descriptive mapping for AI prompt (more natural language)
var ConjugationDescriptions = map[string]string{
	"none":                 "none (no conjugation)",
	"dictionary_form":      "dictionary form",
	"negative":             "negative form",
	"past":                 "past tense",
	"past_negative":        "past negative",
	"te_form":              "te-form (~て)",
	"ta_form":              "ta-form (~た)",
	"masu_form":            "masu-form (polite present)",
	"masu_negative":        "masu negative (ません)",
	"masu_past":            "masu past (ました)",
	"masu_past_negative":   "masu past negative (ませんでした)",
	"imperative":           "imperative (command form)",
	"volitional":           "volitional (let's form)",
	"conditional":          "conditional form",
	"potential":            "potential form (can do)",
	"passive":              "passive form",
	"causative":            "causative form (make someone do)",
	"causative_passive":    "causative passive",
	"progressive":          "progressive form (~ている)",
	"progressive_past":     "progressive past (~ていた)",
	"nagara_form":          "nagara-form (~ながら)",
	"ba_form":              "ba-form (~ば)",
	"tara_form":            "tara-form (~たら)",
	"naide_form":           "naide-form (~ないで)",
	"i_adj_negative":       "i-adjective negative (~くない)",
	"i_adj_past":           "i-adjective past (~かった)",
	"i_adj_past_negative":  "i-adjective past negative (~くなかった)",
	"i_adj_adverb":         "i-adjective adverb form (~く)",
	"na_adj_negative":      "na-adjective negative (~ではない)",
	"na_adj_past":          "na-adjective past (~だった)",
	"na_adj_past_negative": "na-adjective past negative (~ではなかった)",
	"honorific":            "honorific form",
	"humble":               "humble form",
	"auxiliary":            "auxiliary form",
	"particle":             "particle form",
}

// Function to get detailed conjugation list for AI prompts
func GetDetailedConjugationPromptText() string {
	result := ""
	for key, desc := range ConjugationDescriptions {
		result += "\"" + key + "\": " + desc + "\n- "
	}
	if len(result) > 3 {
		result = result[:len(result)-3] // Remove trailing '\n- '
	}
	return result
}
