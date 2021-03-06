package command

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/divjotarora/proxy/bsonutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

const (
	dataDir = "../testdata"
)

func BenchmarkFixers(b *testing.B) {
	listCollsResponse := readJSONFile(b, "list_collections_response.json")

	b.Run("baseline", func(b *testing.B) {
		// Benchmark to get the baseline metrics for creating a new bsoncore.Document by copying every value over
		// without any modifications.
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			iter, err := bsonutil.NewIterator(listCollsResponse)
			if err != nil {
				b.Fatal(err)
			}

			idx, doc := bsoncore.AppendDocumentStart(nil)
			for iter.Next() {
				doc = bsoncore.AppendValueElement(doc, iter.Element().Key(), iter.Value())
			}
			doc, _ = bsoncore.AppendDocumentEnd(doc, idx)
		}
	})
	b.Run("use D", func(b *testing.B) {
		// Benchmark unmarshalling the response to bson.D, fixing, and marshalling the fixed version.
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var unmarshalled bson.D
			err := bson.Unmarshal(listCollsResponse, &unmarshalled)
			if err != nil {
				b.Fatal(err)
			}

			cursorDoc := unmarshalled[0].Value.(bson.D)
			cursorDoc[1].Value = cursorDoc[1].Value.(string)[5:] // Fix cursor.ns value.

			// Fix idIndex.ns value in every batch document.
			batchArray := cursorDoc[2].Value.(bson.A)
			for _, doc := range batchArray {
				doc.(bson.D)[4].Value.(bson.D)[3].Value = doc.(bson.D)[4].Value.(bson.D)[3].Value.(string)[5:]
			}

			_, err = bson.Marshal(unmarshalled)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("use bsoncore", func(b *testing.B) {
		// Benchmark using a DocumentFixer.
		b.ReportAllocs()

		listCollsBatchFixer := DocumentFixer{
			"idIndex": DocumentFixer{
				"ns": ValueFixerFunc(removeDBPrefixValueFixer),
			},
		}
		responseFixer := newDefaultCursorResponseFixer(listCollsBatchFixer)

		for i := 0; i < b.N; i++ {
			_, err := responseFixer.Fix(listCollsResponse)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkPassthrough(b *testing.B) {
	// Simulate a baseline passthrough suite where nothing is being fixed, so the fixer codepath is a no-op.

	listCollsResponse := readJSONFile(b, "list_collections_response.json")

	b.Run("use D", func(b *testing.B) {
		// In the passthrough suite, documents are unmarshalled to bson.D and then immediately re-marshalled.
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var unmarshalled bson.D
			err := bson.Unmarshal(listCollsResponse, &unmarshalled)
			if err != nil {
				b.Fatal(err)
			}

			_, err = bson.Marshal(unmarshalled)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("use bsoncore", func(b *testing.B) {
		// For passthrough, an empty DocumentFixer is used. This will iterate over all values in the document and copy
		// them over using bsoncore.AppendValueElement.
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			df := DocumentFixer{}
			_, err := df.Fix(listCollsResponse)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func readJSONFile(b *testing.B, file string) bsoncore.Document {
	b.Helper()

	path := fmt.Sprintf("%s/%s", dataDir, file)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		b.Fatalf("error reading file path %s: %v", path, err)
	}

	var doc bsoncore.Document
	if err = bson.UnmarshalExtJSON(data, true, &doc); err != nil {
		b.Fatalf("UnmarshalExtJSON error for path %s: %v", path, err)
	}
	return doc
}
