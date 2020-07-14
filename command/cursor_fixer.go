package command

// newCursorResponseFixer creates a DocumentFixer for cursor responses. The provided batchDocsFixer will be called for
// each document in the cursor batch.
func newDefaultCursorResponseFixer(batchDocsFixer ValueFixer) DocumentFixer {
	return DocumentFixer{
		"cursor": newCursorValueFixer(batchDocsFixer),
	}
}

// newCursorResponseFixer creates a ValueFixer for cursor subdocuments. The provided batchDocsFixer will be called for
// each document in the cursor batch.
func newCursorValueFixer(batchDocsFixer ValueFixer) ValueFixer {
	fixers := DocumentFixer{
		"ns": removeDBPrefixValueFixer,
	}
	if batchDocsFixer != nil {
		avf := newArrayValueFixer(batchDocsFixer)
		fixers["firstBatch"] = avf
		fixers["nextBatch"] = avf
	}

	return fixers
}
