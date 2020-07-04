package command

func attachFixers(p *Parser) {
	// listCollections
	listCollResponseFixer := compositeFixer{
		"cursor": newDefaultCursorValueFixer(),
	}
	p.register("listCollections", nil, listCollResponseFixer)
}
