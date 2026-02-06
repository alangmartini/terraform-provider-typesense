# Conversation Model (RAG) Configuration
# Enables conversational search with AI-generated responses based on your data

# =============================================================================
# CONVERSATION HISTORY COLLECTION
# Required by the Conversation Model to store chat history
# =============================================================================
resource "typesense_collection" "conversation_history" {
  count = var.openai_api_key != "" ? 1 : 0

  name = "music_chat_history"

  # Conversation ID to group messages in the same session
  field {
    name = "conversation_id"
    type = "string"
  }

  # Role of the message sender (user or assistant)
  field {
    name = "role"
    type = "string"
  }

  # The actual message content
  field {
    name = "message"
    type = "string"
  }

  # Timestamp for ordering and TTL management
  field {
    name = "timestamp"
    type = "int64"
    sort = true
  }
}

# =============================================================================
# CONVERSATION MODEL (RAG)
# Uses an LLM to answer questions based on your music data
# =============================================================================
resource "typesense_conversation_model" "music_assistant" {
  count = var.openai_api_key != "" ? 1 : 0

  id                 = "music-assistant"
  model_name         = var.conversation_model_name
  api_key            = var.openai_api_key
  history_collection = typesense_collection.conversation_history[0].name

  # System prompt defines the AI assistant's behavior and knowledge domain
  system_prompt = <<-EOT
    You are a helpful music assistant for a digital music store. You have access
    to a comprehensive music catalog including tracks, albums, artists, and playlists.

    Your knowledge covers:
    - Track details: name, duration, composer, genre, price
    - Album information: title, artist, track count, total duration
    - Artist profiles: name, albums, genres they perform
    - Playlist contents: name, tracks, total duration

    When answering questions:
    - Be helpful and conversational
    - Provide specific details when available (e.g., exact song durations, prices)
    - Suggest related music when appropriate
    - If asked about something outside the music catalog, politely redirect

    Example interactions:
    - "What genres does U2 perform?" -> List genres from artist data
    - "How long is the album 'Abbey Road'?" -> Provide total duration
    - "Find me rock songs under $1" -> Search tracks with filters
    - "Tell me about the playlist 'Classical'?" -> Describe playlist contents
  EOT

  # Conversation history TTL: 1 hour (3600 seconds)
  # Shorter TTL for demo purposes; production might use 86400 (24 hours)
  ttl = 3600

  # Limit context size for cost efficiency
  max_bytes = 16000

  depends_on = [typesense_collection.conversation_history]
}
