# Music Genre Synonyms for Chinook Database
# Improves search by treating related terms as equivalent

# =============================================================================
# GENRE SYNONYMS (Multi-way - all terms are equivalent)
# =============================================================================

# Rock music variations
resource "typesense_synonym" "rock" {
  collection = typesense_collection.tracks.name
  name       = "rock-synonyms"
  synonyms   = ["rock", "rock and roll", "rock & roll", "rock n roll"]
}

# R&B and soul variations
resource "typesense_synonym" "rnb" {
  collection = typesense_collection.tracks.name
  name       = "rnb-synonyms"
  synonyms   = ["r&b", "rnb", "rhythm and blues", "rhythm & blues", "soul"]
}

# Hip-hop and rap
resource "typesense_synonym" "hiphop" {
  collection = typesense_collection.tracks.name
  name       = "hiphop-synonyms"
  synonyms   = ["hip-hop", "hip hop", "hiphop", "rap"]
}

# Electronic music variations
resource "typesense_synonym" "electronic" {
  collection = typesense_collection.tracks.name
  name       = "electronic-synonyms"
  synonyms   = ["electronic", "electronica", "edm", "dance"]
}

# Classical music
resource "typesense_synonym" "classical" {
  collection = typesense_collection.tracks.name
  name       = "classical-synonyms"
  synonyms   = ["classical", "classic", "orchestra", "orchestral", "symphonic"]
}

# Metal variations
resource "typesense_synonym" "metal" {
  collection = typesense_collection.tracks.name
  name       = "metal-synonyms"
  synonyms   = ["metal", "heavy metal", "hard rock"]
}

# Country and folk
resource "typesense_synonym" "country" {
  collection = typesense_collection.tracks.name
  name       = "country-synonyms"
  synonyms   = ["country", "country western", "folk", "americana"]
}

# Jazz variations
resource "typesense_synonym" "jazz" {
  collection = typesense_collection.tracks.name
  name       = "jazz-synonyms"
  synonyms   = ["jazz", "smooth jazz", "bebop", "swing"]
}

# Latin music
resource "typesense_synonym" "latin" {
  collection = typesense_collection.tracks.name
  name       = "latin-synonyms"
  synonyms   = ["latin", "latino", "salsa", "bossa nova", "reggaeton"]
}

# Blues variations
resource "typesense_synonym" "blues" {
  collection = typesense_collection.tracks.name
  name       = "blues-synonyms"
  synonyms   = ["blues", "rhythm blues", "delta blues"]
}

# Pop music
resource "typesense_synonym" "pop" {
  collection = typesense_collection.tracks.name
  name       = "pop-synonyms"
  synonyms   = ["pop", "popular", "top 40", "mainstream"]
}

# Reggae variations
resource "typesense_synonym" "reggae" {
  collection = typesense_collection.tracks.name
  name       = "reggae-synonyms"
  synonyms   = ["reggae", "ska", "dub", "dancehall"]
}

# =============================================================================
# MEDIA TYPE SYNONYMS (One-way - map alternatives to canonical term)
# =============================================================================

# Digital audio formats
resource "typesense_synonym" "mp3" {
  collection = typesense_collection.tracks.name
  name       = "mp3-synonyms"
  root       = "MPEG audio file"
  synonyms   = ["mp3", "mpeg", "digital audio"]
}

# Protected audio formats
resource "typesense_synonym" "aac" {
  collection = typesense_collection.tracks.name
  name       = "aac-synonyms"
  root       = "AAC audio file"
  synonyms   = ["aac", "m4a", "apple audio"]
}

# Video formats
resource "typesense_synonym" "video" {
  collection = typesense_collection.tracks.name
  name       = "video-synonyms"
  root       = "MPEG-4 video file"
  synonyms   = ["video", "mp4", "movie", "film"]
}

# =============================================================================
# ARTIST SEARCH SYNONYMS (Applied to albums collection too)
# =============================================================================

# The artist synonyms help when searching for bands/artists with variations
resource "typesense_synonym" "acdc_albums" {
  collection = typesense_collection.albums.name
  name       = "acdc-synonyms"
  synonyms   = ["ac/dc", "acdc", "ac dc"]
}

resource "typesense_synonym" "acdc_tracks" {
  collection = typesense_collection.tracks.name
  name       = "acdc-track-synonyms"
  synonyms   = ["ac/dc", "acdc", "ac dc"]
}

# =============================================================================
# PLAYLIST SEARCH SYNONYMS
# =============================================================================

# Genre synonyms for playlist searches
resource "typesense_synonym" "playlist_rock" {
  collection = typesense_collection.playlists.name
  name       = "playlist-rock-synonyms"
  synonyms   = ["rock", "rock and roll", "rock & roll"]
}

resource "typesense_synonym" "playlist_hiphop" {
  collection = typesense_collection.playlists.name
  name       = "playlist-hiphop-synonyms"
  synonyms   = ["hip-hop", "hip hop", "hiphop", "rap"]
}

resource "typesense_synonym" "playlist_classical" {
  collection = typesense_collection.playlists.name
  name       = "playlist-classical-synonyms"
  synonyms   = ["classical", "classic", "orchestra", "orchestral"]
}
