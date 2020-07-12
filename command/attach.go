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
	listCollsCursorFixer := newCursorValueFixer(listCollsBatchFixer)
	p.register("listCollections", nil, compositeFixer{"cursor": listCollsCursorFixer})
}
