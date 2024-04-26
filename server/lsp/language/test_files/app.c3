module app;

char ambiguousVariable = 'C';
const int BACKGROUND = 3;
const int SOMETHING_HAPPENED = 4;
enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
fault WindowError { UNEXPECTED_ERROR, SOMETHING_HAPPENED }

fn void run(int tick) {
    tick = tick + 3;
    WindowStatus status;
    status = WindowStatus.BACKGROUND;
}

fn void main() {
    int ambiguousVariable = 3;
    WindowError error = WindowError.SOMETHING_HAPPENED;

    showHelp();
    helpDisplayedTimes = 1;
    run(3);
}

fn void calling_main() {
    main();
}