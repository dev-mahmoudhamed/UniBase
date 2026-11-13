package utils

import "testing"

func TestCellValue(t *testing.T) {
	guid := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	if got := cellValue(guid, "UNIQUEIDENTIFIER"); got != "03020100-0504-0706-0809-0A0B0C0D0E0F" {
		t.Errorf("guid: got %v", got)
	}
	if got := cellValue([]byte("café"), "VARCHAR"); got != "café" {
		t.Errorf("text: got %v", got)
	}
	if got := cellValue([]byte{0xff, 0x00}, "BYTEA"); got != "0xff00" {
		t.Errorf("binary: got %v", got)
	}
	if got := cellValue(int64(3), "INT8"); got != int64(3) {
		t.Errorf("non-bytes pass through: got %v", got)
	}
}

func TestSplitSQLBatches(t *testing.T) {
	got := SplitSQLBatches("SELECT 1\nGO\n\ngo\nSELECT 2")
	if len(got) != 2 || got[0] != "SELECT 1" || got[1] != "SELECT 2" {
		t.Errorf("got %q", got)
	}
}
