fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }

fn void main() {
    WindowError error = WindowError.SOMETHING_HAPPENED;
    error = UNEXPECTED_ERROR;
}