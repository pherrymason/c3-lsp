module emulator;

interface EmulatorConsole
{
    fn void run();
}

struct Emu (EmulatorConsole) {
  Cpu c;
  Audio a;
  bool on;
}

fn Emu newEmulator(){
	Emu emulator;
	emulator.on = true;
	return emulator;
}

fn void Emu.run() {
    
}