resource "typesense_stemming_dictionary" "music_terms" {
  dictionary_id = "music-terms"

  words = [
    { word = "guitars", stem = "guitar" },
    { word = "drumming", stem = "drum" },
    { word = "singing", stem = "sing" },
    { word = "recordings", stem = "recording" },
  ]
}
