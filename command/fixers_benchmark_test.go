package command

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/divjotarora/proxy/bsonutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	mgobson "gopkg.in/mgo.v2/bson"
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
			var unmarshalled mgobson.D
			err := mgobson.Unmarshal(listCollsResponse, &unmarshalled)
			if err != nil {
				b.Fatal(err)
			}

			cursorDoc := unmarshalled[0].Value.(mgobson.D)
			cursorDoc[1].Value = cursorDoc[1].Value.(string)[5:] // Fix cursor.ns value.

			// Fix idIndex.ns value in every batch document.
			batchArray := cursorDoc[2].Value.([]interface{})
			for _, doc := range batchArray {
				doc.(mgobson.D)[4].Value.(mgobson.D)[3].Value = doc.(mgobson.D)[4].Value.(mgobson.D)[3].Value.(string)[5:]
			}

			_, err = mgobson.Marshal(unmarshalled)
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
