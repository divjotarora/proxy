package command

func attachFixers(p *Parser) {
	// listCollections: cursor subdocument where each batch document is in the form
	// {name: <collName>, ..., idIndex: {ns: <coll namespace>, ...}}
	// The idIndex.ns value in each batch document needs to be fixed to remove the DB prefix.
	listCollsBatchFixer := DocumentFixer{
		"idIndex": DocumentFixer{
			"ns": ValueFixerFunc(removeDBPrefixValueFixer),
		},
	}
	listCollsResponseFixer := newDefaultCursorResponseFixer(listCollsBatchFixer)
	p.register("listCollections", nil, listCollsResponseFixer)

	// listIndexes: each batch document has an ns value.
	listIndexesBatchFixer := DocumentFixer{
		"ns": ValueFixerFunc(removeDBPrefixValueFixer),
	}
	listIndexesResponseFixer := newDefaultCursorResponseFixer(listIndexesBatchFixer)
	p.register("listIndexes", nil, listIndexesResponseFixer)

	// find: simple cursor subdocument.
	findResponseFixer := newDefaultCursorResponseFixer(nil)
	p.register("find", nil, findResponseFixer)
}
