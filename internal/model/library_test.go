package model

import (
	"reflect"
	"testing"
)

func TestBookAssociationsHaveDatabaseAndJSONContracts(t *testing.T) {
	bookType := reflect.TypeOf(Book{})
	category, ok := bookType.FieldByName("Category")
	if !ok || category.Tag.Get("gorm") == "" || category.Tag.Get("json") != "category" {
		t.Fatal("Book.Category must expose the category association")
	}
	authors, ok := bookType.FieldByName("Authors")
	if !ok || authors.Tag.Get("gorm") != "many2many:book_authors;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" {
		t.Fatalf("Book.Authors has unexpected GORM association tag: %q", authors.Tag.Get("gorm"))
	}
}

func TestCopyStatuses(t *testing.T) {
	statuses := []string{CopyAvailable, CopyCheckedOut, CopyLost}
	want := []string{"available", "checked_out", "lost"}
	if !reflect.DeepEqual(statuses, want) {
		t.Fatalf("copy statuses = %v, want %v", statuses, want)
	}
}
