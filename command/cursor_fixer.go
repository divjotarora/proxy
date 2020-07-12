package command

// newCursorResponseFixer creates a DocumentFixer for cursor responses. The provided batchDocsFixer will be called for
// each document in the cursor batch.
func newCursorResponseFixer(batchDocsFixer ValueFixer) DocumentFixer {
	fixers := DocumentFixer{
		"ns": removeDBPrefixValueFixer,
	}
	if batchDocsFixer != nil {
		avf := newArrayValueFixer(batchDocsFixer)
		fixers["firstBatch"] = avf
		fixers["nextBatch"] = avf
	}

	return DocumentFixer{
		"cursor": newDocumentValueFixer(fixers),
	}
}
