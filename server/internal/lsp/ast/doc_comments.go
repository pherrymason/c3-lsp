package ast

type DocCommentContract struct {
	name string
	body string
}

type DocComment struct {
	Body      string
	Contracts []*DocCommentContract
}

// Creates a doc comment with the given body.
func NewDocComment(body string) *DocComment {
	return &DocComment{
		Body:      body,
		Contracts: []*DocCommentContract{},
	}
}

// Creates a contract with the given name and body.
// It is expected that the name begins with '@'.
func NewDocCommentContract(name string, body string) *DocCommentContract {
	return &DocCommentContract{
		name,
		body,
	}
}

// Add contracts to the given doc comment.
func (d *DocComment) AddContracts(contracts []*DocCommentContract) {
	d.Contracts = append(d.Contracts, contracts...)
}

func (d *DocComment) GetBody() string {
	return d.Body
}

// Return a string displaying the body and contracts as markdown.
func (d *DocComment) DisplayBodyWithContracts() string {
	out := d.Body

	for _, c := range d.Contracts {
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
