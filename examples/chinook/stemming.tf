resource "typesense_stemming_dictionary" "music_terms" {
  dictionary_id = "music-terms"

  words {
    word = "guitars"
    stem = "guitar"
  }
  words {
    word = "drumming"
    stem = "drum"
  }
  words {
    word = "singing"
    stem = "sing"
  }
  words {
    word = "recordings"
    stem = "recording"
  }
}
