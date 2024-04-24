module machine;
struct Emu {
  Cpu c;
  Audio a;
  bool on;
}
enum WindowStatus { OPEN, BACKGROUND, MINIMIZED }
fn Emu app() {
	Emu emulator;
    emulator.on = true;
	run(3);
	return emulator;
}
fn void run(int tick) {
	tick = tick + 3;
	WindowStatus status;
	status = BACKGROUND;
}