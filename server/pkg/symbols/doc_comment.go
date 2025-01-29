package symbols

type DocCommentContract struct {
	name string
	body string
}

type DocComment struct {
	body      string
	contracts []*DocCommentContract
}

// Creates a doc comment with the given body.
func NewDocComment(body string) DocComment {
	return DocComment{
		body:      body,
		contracts: []*DocCommentContract{},
	}
}

// Creates a contract with the given name and body.
// It is expected that the name begins with '@'.
func NewDocCommentContract(name string, body string) DocCommentContract {
	return DocCommentContract{
		name,
		body,
	}
}

// Add contracts to the given doc comment.
func (d *DocComment) AddContracts(contracts []*DocCommentContract) {
	d.contracts = append(d.contracts, contracts...)
}

func (d *DocComment) HasContracts() bool {
	return len(d.contracts) > 0
}

func (d *DocComment) GetContracts() []*DocCommentContract {
	return d.contracts
}

func (d *DocComment) GetBody() string {
	return d.body
}

func (c *DocCommentContract) GetName() string {
	return c.name
}

func (c *DocCommentContract) GetBody() string {
	return c.body
}

// Return a string displaying the body and contracts as markdown.
func (d *DocComment) DisplayBodyWithContracts() string {
	out := d.body

	for _, c := range d.contracts {
		if out != "" {
			out += "\n\n"
		}
		out += "**" + c.name + "**"
		if c.body != "" {
			out += " " + c.body
		}
	}

	return out
}

type DocCommentBuilder struct {
	docComment DocComment
}

func NewDocCommentBuilder(body string) *DocCommentBuilder {
	return &DocCommentBuilder{
		docComment: NewDocComment(body),
	}
}

func (b *DocCommentBuilder) WithContract(name string, body string) *DocCommentBuilder {
	contract := NewDocCommentContract(name, body)
	b.docComment.contracts = append(b.docComment.contracts, &contract)
	return b
}

func (b *DocCommentBuilder) Build() DocComment {
	return b.docComment
}
