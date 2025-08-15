package server

// SchemaString contains the GraphQL schema definition as a string.
const SchemaString = `
  type PartOfSpeech {
    id: Int!
    pos: String
    eng: String
    abr: String
    furigana: String
    created_at: String!
    words(limit: Int = 50, offset: Int = 0): [Word!]!
    wordCount: Int!
  }

  type Grammar {
    id: Int!
    name: String!
    description: String
    example: String
    sentenceWords(limit: Int = 50, offset: Int = 0): [SentenceWord!]!
    usageCount: Int!
  }

  type Word {
    id: Int!
    lemma: String!
    lemma_furigana: String!
    part_of_speech_id: Int!
    confidence: Float
    ai: Boolean!
    frequency_rank: Int
    jlpt_level: Int
    updated_at: String
    partOfSpeech: PartOfSpeech!
    sentenceWords: [SentenceWord!]!
    sentences: [Sentence!]!
  }

  type ArticleSentence {
    id: Int!
    part: String!
    sub_index: Int!
    position: Int!
    media_url: String
    sentence: Sentence!
    article_id: Int!
  }

  type Sentence {
    id: Int!
    jp: String!
    en: String!
    eval: String!
    sentenceWords: [SentenceWord!]!
    words: [Word!]!
    articles: [Article!]!
    wordCount: Int!
  }

  type SentenceWord {
    id: Int!
    sentence_id: Int!
    word_id: Int!
    position: Int
    jp: String
    furigana: String
    conjugation: String
    grammar_id: Int
    contextual_meaning: [String]
    confidence: Float
    sentence: Sentence!
    word: Word!
    grammar: Grammar
  }

  type Article {
    id: Int!
    category: Category
    category_id: Int
    source_url: String!
    date: String
    video: String
    created_at: String
    processed_at: String
    article_sentences: [ArticleSentence!]!
  }

  type Category {
    id: Int!
    created_at: String
    category_english: String
    category_japanese: String
    articleLinks(limit: Int = 10, offset: Int = 0): [ArticleLink!]!
    articleLinkCount: Int!
  }

  type Link {
    id: Int!
    category_id: Int!
    title: String!
    description: String
    link: String!
    url: String!
    guid: String!
    pubDate: String
    preview: Boolean
    preview_img_link: String
    created_at: String
    translated: Boolean
    translated_date: String
    category: Category!
  }

  type Query {
    partsOfSpeech: [PartOfSpeech!]!
    grammar: [Grammar!]!
    words(limit: Int = 10, offset: Int = 0): [Word!]!
    word(id: Int!): Word
    sentences(limit: Int = 10, offset: Int = 0): [Sentence!]!
    sentence(id: Int!): Sentence
    sentenceWords(limit: Int = 20, offset: Int = 0): [SentenceWord!]!
    categories: [Category!]!
    articles(limit: Int = 10, offset: Int = 0, categoryId: Int): [Article!]!
    links(limit: Int = 10, offset: Int = 0, categoryId: Int): [Link!]!
    article(id: Int!): Article
    ArticleSentence(id: Int!): ArticleSentence
    articleIdByUrl(url: String!): Int
    hello: String!
    articleByUrl(url: String!): Article
  }

  type Mutation {
    _empty: String
  }
`
