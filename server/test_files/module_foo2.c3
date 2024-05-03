module foo2;
import foo::bar;
import cyclic;
import foo::triangle;

fn void main() {
    Bar mybar;
    mybar.weight = BAR_WEIGHT;
    mybar.color = foo::bar::DEFAULT_BAR_COLOR;
    Triangle triangle;
}