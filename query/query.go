package query

type FilterFunction func(metadata []byte) (bool, error)

// FilterFunctionFromQuery takes a query string and returns a FilterFunction.
// This function performs the following steps:
// 1. Lexical analysis
// 2. Parsing
// 3. AST compilation
// 4. Filter function creation
func FilterFunctionFromQuery(query string) (FilterFunction, error) {
	// Create a new lexer with the input query string
	lexer := NewLexer(query)

	// Create a new parser using the lexer
	parser := NewParser(lexer)

	// Parse the query and generate an Abstract Syntax Tree (AST)
	ast, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	// Compile the AST into a CompiledExpression
	compiledExpr := CompileExpression(ast)

	// Create a filter function using the compiled expression
	filterFunc := CreateFilterFunction(compiledExpr)

	// Return the filter function and any error that occurred during the process
	return filterFunc, err
}
