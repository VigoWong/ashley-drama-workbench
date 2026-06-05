package tools

import "testing"

func TestCatalogReturnsAllWhenNoFilter(t *testing.T) {
	all := GetProductCatalog("")
	if len(all) < 4 {
		t.Fatalf("expected several categories, got %d", len(all))
	}
}

func TestCatalogFiltersByCategory(t *testing.T) {
	got := GetProductCatalog("sofa")
	if len(got) == 0 {
		t.Fatal("expected a sofa match")
	}
	for _, p := range got {
		if p.Category != "sofa" {
			t.Fatalf("unexpected category %s", p.Category)
		}
	}
}

func TestTropesReturnsHomeVertical(t *testing.T) {
	tr := GetWinningTropes("US", "home")
	if len(tr) == 0 {
		t.Fatal("expected home tropes")
	}
}
