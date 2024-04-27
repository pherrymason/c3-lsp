module emulator;

interface EmulatorConsole
{
    fn void run();
}

struct Audio {
    int frequency;
}

fn void Audio.init() {

}

struct Emu (EmulatorConsole) {
  Cpu cpu;
  Audio a;
  bool on;
}

fn Emu newEmulator(){
	Emu emulator;
    emulator.cpu.init();
    emulator.audio.init();
	emulator.on = true;
	return emulator;
}

fn void Emu.run() {
    
}

fn void Emu.blarg(Emu* emu) {
    emu.cpu.init();
    emu.audio.init();
}