module emulator::cpu;

const int CPU_FREQUENCY = 66;

struct Cpu {
	int tick;
}
fn Cpu NewCpu() {
	Cpu c;
	return c;
}
