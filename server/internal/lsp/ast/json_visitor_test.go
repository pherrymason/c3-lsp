package ast

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestJSONVisitor_overall(t *testing.T) {
	source := `module foo;
	int hello = 0;
	fn void main(){
		fmt.print("hero", hello);
	}`

	ast := ConvertToAST(GetCST(source), source, "file.c3")

	visitor := JSONVisitor{}
	Visit(&ast, &visitor)

	jsonString, err := visitor.ToJSONString()
	if err != nil {
		fmt.Println("Error al converting to JSON:", err)
		return
	}
	fmt.Printf("%s", jsonString)
	data := []byte(jsonString)
	err = os.WriteFile("./ast.json", data, 0777)
	if err != nil {
		log.Fatal(err)
	}
}
