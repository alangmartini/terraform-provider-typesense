# Natural Language Search Model Configuration
# Enables natural language queries like "rock songs longer than 5 minutes"
# to be automatically converted into structured Typesense filters

# The NL search model uses an LLM to parse natural language queries
# and convert them into filter_by, sort_by, and q parameters
resource "typesense_nl_search_model" "music_search" {
  count = var.openai_api_key != "" ? 1 : 0

  id         = "music-nl-search"
  model_name = var.nl_model_name
  api_key    = var.openai_api_key

  # Domain-specific instructions to help the LLM understand the music data
  system_prompt = <<-EOT
    You are parsing natural language queries for a music database.

    Available fields for filtering:
    - genre_name: Music genre (Rock, Jazz, Pop, Metal, Classical, etc.)
    - artist_name: Name of the artist/band
    - album_title: Name of the album
    - unit_price: Price of the track (float, typically 0.99 or 1.99)
    - milliseconds: Duration of the track in milliseconds (use for "long songs", "short tracks")

    Example interpretations:
    - "rock songs" → filter_by: genre_name:=Rock
    - "songs by U2" → filter_by: artist_name:=U2
    - "tracks longer than 5 minutes" → filter_by: milliseconds:>300000
    - "cheap rock songs" → filter_by: genre_name:=Rock && unit_price:<1.00
    - "classical music sorted by duration" → filter_by: genre_name:=Classical, sort_by: milliseconds:desc
  EOT

  # Conservative temperature for consistent, predictable filter generation
  temperature = 0.0

  # Limit payload size to stay within typical context windows
  max_bytes = 16000
}
