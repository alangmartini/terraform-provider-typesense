# Stopwords Sets for Chinook Music Database
# Filters out common words during search to improve relevance

# =============================================================================
# ENGLISH STOPWORDS
# Common articles, prepositions, and conjunctions that don't add search value
# =============================================================================
resource "typesense_stopwords_set" "english_common" {
  name   = "english-common"
  locale = "en"

  stopwords = [
    # Articles
    "a", "an", "the",

    # Prepositions
    "at", "by", "for", "from", "in", "of", "on", "to", "with",

    # Conjunctions
    "and", "but", "or",

    # Common filler words
    "is", "it", "be", "are", "was", "were", "been",
  ]
}

# =============================================================================
# MUSIC-SPECIFIC STOPWORDS
# Industry terms that appear frequently but don't help narrow searches
# =============================================================================
resource "typesense_stopwords_set" "music_terms" {
  name   = "music-terms"
  locale = "en"

  stopwords = [
    # Featuring variations (handled by synonyms for mapping, stopwords for filtering)
    "ft", "feat",

    # Version/remix indicators (often want exact match anyway)
    "mix",

    # Common music qualifiers
    "live", "acoustic", "unplugged",

    # Track numbering artifacts
    "track", "disc", "cd",
  ]
}

# =============================================================================
# INVOICE/BILLING STOPWORDS
# Common terms in billing addresses that don't improve search
# =============================================================================
resource "typesense_stopwords_set" "billing_terms" {
  name   = "billing-terms"
  locale = "en"

  stopwords = [
    # Address components
    "st", "street", "ave", "avenue", "rd", "road", "blvd", "boulevard",
    "dr", "drive", "ln", "lane", "ct", "court", "pl", "place",

    # Unit indicators
    "apt", "apartment", "suite", "ste", "unit", "floor", "fl",

    # Directional
    "n", "s", "e", "w", "north", "south", "east", "west",
    "ne", "nw", "se", "sw",
  ]
}
