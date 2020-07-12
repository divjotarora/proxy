package command

func newCursorResponseFixer(batchDocsFixer valueFixer) compositeFixer {
	fixers := compositeFixer{
		"ns": removeDBPrefixValueFixer,
	}
	if batchDocsFixer != nil {
		avf := newArrayValueFixer(batchDocsFixer)
		fixers["firstBatch"] = avf
		fixers["nextBatch"] = avf
	}

	return compositeFixer{
		"cursor": newDocumentValueFixer(fixers),
	}
}
