package command

func attachFixers(p *Parser) {
	// listCollections
	listCollResponseFixer := compositeFixer{
		"cursor": newCursorValueFixer(nil),
	}
	p.register("listCollections", nil, listCollResponseFixer)
}
