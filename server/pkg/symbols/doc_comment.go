package symbols

type DocCommentContract struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

type DocComment struct {
	Body      string                `json:"body"`
	Contracts []*DocCommentContract `json:"contracts"`
}

// Creates a doc comment with the given body.
func NewDocComment(body string) DocComment {
	return DocComment{
		Body:      body,
		Contracts: []*DocCommentContract{},
	}
}

// Creates a contract with the given name and body.
// It is expected that the name begins with '@'.
func NewDocCommentContract(name string, body string) DocCommentContract {
	return DocCommentContract{
		Name: name,
		Body: body,
	}
}

// Add contracts to the given doc comment.
func (d *DocComment) AddContracts(contracts []*DocCommentContract) {
	d.Contracts = append(d.Contracts, contracts...)
}

func (d *DocComment) HasContracts() bool {
	return len(d.Contracts) > 0
}

func (d *DocComment) GetContracts() []*DocCommentContract {
	return d.Contracts
}

func (d *DocComment) GetBody() string {
	return d.Body
}

func (c *DocCommentContract) GetName() string {
	return c.Name
}

func (c *DocCommentContract) GetBody() string {
	return c.Body
}

// Return a string displaying the body and contracts as markdown.
func (d *DocComment) DisplayBodyWithContracts() string {
	out := d.Body

	for _, c := range d.Contracts {
		if out != "" {
			out += "\n\n"
		}
		out += "**" + c.Name + "**"
		if c.Body != "" {
			out += " " + c.Body
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
	b.docComment.Contracts = append(b.docComment.Contracts, &contract)
	return b
}

func (b *DocCommentBuilder) Build() DocComment {
	return b.docComment
}
