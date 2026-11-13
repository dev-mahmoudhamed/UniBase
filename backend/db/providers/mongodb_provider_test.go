package providers

import (
	"encoding/json"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestToRelaxedDocKeepsTypes(t *testing.T) {
	raw, _ := bson.Marshal(bson.D{
		{Key: "tags", Value: bson.A{"a", "b"}},
		{Key: "price", Value: 10.0},
		{Key: "qty", Value: int32(5)},
		{Key: "id", Value: primitive.NilObjectID},
	})
	doc, err := toRelaxedDoc(bson.Raw(raw))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := json.Marshal(doc)
	want := `{"id":{"$oid":"000000000000000000000000"},"price":10.0,"qty":5,"tags":["a","b"]}`
	if string(got) != want {
		t.Fatalf("got %s\nwant %s", got, want)
	}
}

func TestKeepNumericType(t *testing.T) {
	if v := keepNumericType(bsontype.Double, int32(12)); v != float64(12) {
		t.Errorf("double field: got %T", v)
	}
	if v := keepNumericType(bsontype.Int64, int32(7)); v != int64(7) {
		t.Errorf("int64 field: got %T", v)
	}
	if v := keepNumericType(bsontype.Int32, int32(7)); v != int32(7) {
		t.Errorf("int32 field: got %T", v)
	}
	if v := keepNumericType(bsontype.String, "x"); v != "x" {
		t.Errorf("string passes through: got %v", v)
	}
}
