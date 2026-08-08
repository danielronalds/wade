package reviewsnapshots

import (
	"reflect"
	"testing"
)

func TestParseNameStatusZ(t *testing.T) {
	output := []byte("M\x00src/app.ts\x00A\x00new file.ts\x00D\x00old.txt\x00R100\x00from name.txt\x00to name.txt\x00")

	got := parseNameStatusZ(output)
	want := []changedPath{
		{status: StatusModified, oldPath: stringPtr("src/app.ts"), newPath: stringPtr("src/app.ts")},
		{status: StatusAdded, oldPath: nil, newPath: stringPtr("new file.ts")},
		{status: StatusDeleted, oldPath: stringPtr("old.txt"), newPath: nil},
		{status: StatusRenamed, oldPath: stringPtr("from name.txt"), newPath: stringPtr("to name.txt")},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseNameStatusZ() = %#v, want %#v", got, want)
	}
}

func TestFilterReviewablePathsExcludesBinaryAndMinifiedFiles(t *testing.T) {
	paths := []string{"src/app.ts", "dist/app.min.js", "assets/logo.png", "styles/site.css"}
	got := filterReviewablePaths(paths)
	want := []string{"src/app.ts", "styles/site.css"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterReviewablePaths() = %#v, want %#v", got, want)
	}
}
