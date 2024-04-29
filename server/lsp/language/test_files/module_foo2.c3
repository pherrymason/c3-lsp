module foo2;
import foo::bar;

fn void main() {
    Bar mybar;
    mybar.weight = BAR_WEIGHT;
    mybar.color = foo::bar::DEFAULT_BAR_COLOR;
}