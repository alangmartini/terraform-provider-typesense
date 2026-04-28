package migrator

import "testing"

func TestImportDocumentsURLEscapesCollectionName(t *testing.T) {
	got := importDocumentsURL("http://127.0.0.1:8108/", "docs / prod")
	want := "http://127.0.0.1:8108/collections/docs%20%2F%20prod/documents/import?action=upsert"
	if got != want {
		t.Fatalf("importDocumentsURL() = %q, want %q", got, want)
	}
}
