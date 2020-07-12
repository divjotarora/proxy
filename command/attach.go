package command

func attachFixers(p *Parser) {
	// listCollections: cursor subdocument where each batch document is in the form
	// {name: <collName>, ..., idIndex: {ns: <coll namespace>, ...}}
	// The idIndex.ns value in each batch document needs to be fixed to remove the DB prefix.
	listCollsBatchFixer := newDocumentValueFixer(compositeFixer{
		"idIndex": newDocumentValueFixer(compositeFixer{
			"ns": valueFixerFunc(removeDBPrefixValueFixer),
		}),
	})
	listCollsResponseFixer := newCursorResponseFixer(listCollsBatchFixer)
	p.register("listCollections", nil, listCollsResponseFixer)

	// listIndexes: each batch document has an ns value.
	listIndexesBatchFixer := newDocumentValueFixer(compositeFixer{
		"ns": valueFixerFunc(removeDBPrefixValueFixer),
	})
	listIndexesResponseFixer := newCursorResponseFixer(listIndexesBatchFixer)
	p.register("listIndexes", nil, listIndexesResponseFixer)

	// find: simple cursor subdocument.
	findResponseFixer := newCursorResponseFixer(nil)
	p.register("find", nil, findResponseFixer)
}
