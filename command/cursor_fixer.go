package command

func newCursorValueFixer(batchDocsFixer valueFixer) valueFixer {
	fixers := compositeFixer{
		"ns": valueFixerFunc(removeDBPrefixValueFixer),
	}
	if batchDocsFixer != nil {
		avf := newArrayValueFixer(batchDocsFixer)
		fixers["firstBatch"] = avf
		fixers["nextBatch"] = avf
	}

	return newDocumentValueFixer(fixers)
}
