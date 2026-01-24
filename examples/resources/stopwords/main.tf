# Example: Typesense Stopwords Sets

terraform {
  required_providers {
    typesense = {
      source  = "alanm/typesense"
      version = "~> 0.1"
    }
  }
}

provider "typesense" {
  server_host    = var.typesense_host
  server_api_key = var.typesense_api_key
}

variable "typesense_host" {
  type        = string
  description = "Typesense server hostname"
}

variable "typesense_api_key" {
  type        = string
  description = "Typesense server API key"
  sensitive   = true
}

# English stopwords set
resource "typesense_stopwords_set" "english" {
  name   = "english-stopwords"
  locale = "en"
  stopwords = [
    "a", "an", "and", "are", "as", "at", "be", "but", "by",
    "for", "if", "in", "into", "is", "it", "no", "not", "of",
    "on", "or", "such", "that", "the", "their", "then", "there",
    "these", "they", "this", "to", "was", "will", "with"
  ]
}

# German stopwords set
resource "typesense_stopwords_set" "german" {
  name   = "german-stopwords"
  locale = "de"
  stopwords = [
    "aber", "als", "am", "an", "auch", "auf", "aus", "bei",
    "bin", "bis", "bist", "da", "dadurch", "daher", "darum",
    "das", "dass", "dein", "deine", "dem", "den", "der", "des",
    "die", "dies", "dieser", "du", "durch", "ein", "eine", "einem",
    "einen", "einer", "er", "es", "euer", "eure", "für", "hatte",
    "hatten", "hattest", "hattet", "hier", "ich", "ihr", "ihre",
    "im", "in", "ist", "ja", "jede", "jedem", "jeden", "jeder",
    "jedes", "jener", "kann", "kannst", "können", "könnt", "machen",
    "mein", "meine", "mit", "muß", "mußt", "musst", "müssen", "müßt",
    "nach", "nachdem", "nein", "nicht", "nun", "oder", "seid", "sein",
    "seine", "sich", "sie", "sind", "soll", "sollen", "sollst", "sollt",
    "sonst", "soweit", "sowie", "und", "unser", "unsere", "unter", "vom",
    "von", "vor", "wann", "warum", "was", "weiter", "weitere", "wenn",
    "wer", "werde", "werden", "werdet", "weshalb", "wie", "wieder",
    "wieso", "wir", "wird", "wirst", "wo", "woher", "wohin", "zu",
    "zum", "zur", "über"
  ]
}

# French stopwords set
resource "typesense_stopwords_set" "french" {
  name   = "french-stopwords"
  locale = "fr"
  stopwords = [
    "a", "ai", "aie", "aient", "aies", "ait", "as", "au", "aura",
    "aurai", "auraient", "aurais", "aurait", "auras", "aurez", "auriez",
    "aurions", "aurons", "auront", "aux", "avaient", "avais", "avait",
    "avec", "avez", "aviez", "avions", "avons", "ayant", "ayez", "ayons",
    "c", "ce", "ceci", "cela", "ces", "cet", "cette", "d", "dans", "de",
    "des", "du", "elle", "elles", "en", "es", "est", "et", "eu", "eue",
    "eues", "eurent", "eus", "eusse", "eussent", "eusses", "eussiez",
    "eussions", "eut", "eux", "furent", "fus", "fusse", "fussent", "fusses",
    "fussiez", "fussions", "fut", "il", "ils", "j", "je", "l", "la", "le",
    "les", "leur", "leurs", "lui", "m", "ma", "mais", "me", "mes", "moi",
    "mon", "n", "ne", "nos", "notre", "nous", "on", "ont", "ou", "par",
    "pas", "pour", "qu", "que", "quel", "quelle", "quelles", "quels", "qui",
    "s", "sa", "sans", "se", "sera", "serai", "seraient", "serais", "serait",
    "seras", "serez", "seriez", "serions", "serons", "seront", "ses", "soi",
    "soient", "sois", "soit", "sommes", "son", "sont", "soyez", "soyons",
    "suis", "sur", "t", "ta", "te", "tes", "toi", "ton", "tu", "un", "une",
    "vos", "votre", "vous", "y"
  ]
}

# Custom e-commerce stopwords
resource "typesense_stopwords_set" "ecommerce" {
  name = "ecommerce-stopwords"
  stopwords = [
    "buy", "shop", "purchase", "order", "cart", "checkout",
    "price", "cost", "cheap", "expensive", "deal", "sale",
    "discount", "coupon", "promo", "offer", "shipping", "delivery",
    "free", "fast", "best", "top", "good", "great", "amazing"
  ]
}

# Minimal stopwords for precise search
resource "typesense_stopwords_set" "minimal" {
  name   = "minimal-stopwords"
  locale = "en"
  stopwords = [
    "a", "an", "the"
  ]
}
