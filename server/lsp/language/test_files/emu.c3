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
  Audio audio;
  bool on;
}

fn Emu newEmulator(){
	Emu emulator;
    emulator.cpu.init();
    emulator.audio.init();
	emulator.on = true;
	return emulator;
}

fn void Emu.run(Emu* emu) {
    emu.tick();
}

fn void Emu.blarg(Emu* emu) {
    emu.cpu.init();
    emu.audio.init();
}

fn void Emu.tick(Emu* emu) {
    
}