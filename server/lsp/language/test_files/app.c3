module app;

enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }

fn void run(int tick) {
	tick = tick + 3;
	WindowStatus status;
	status = BACKGROUND;
}

fn void main() {
    showHelp();
    run(3);
}