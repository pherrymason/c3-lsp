module emulator;

struct Emu {
  Cpu c;
  Audio a;
  bool on;
}

fn Emu newEmulator(){
	Emu emulator;
	emulator.on = true;
	return emulator;
}
