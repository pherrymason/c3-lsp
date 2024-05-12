module structs_emulator;

interface EmulatorConsole
{
    fn void run();
}

Emu emulator; // Trap for ""
Cpu cpu; // Trap for finding struct member when cursor is on declaration member.

struct Emu (EmulatorConsole) {
  Cpu cpu;
  Audio audio;
  bool on;
}

fn Emu newEmu() {
    Emu emulator;
    emulator.on = true;
    return emulator;
}

fn void Emu.run(Emu* emu) {
    emulator.on = true;
    emu.tick();
}

fn void Emu.init(Emu* emu) {
    emu.cpu.init();
    emu.audio.init();
    emu.audio.unknown();
    Audio.init(&emu.audio); // Another way to call emu.audio.init()
}


fn void main() {
    Emu emulator;
    emulator.init();
    Emu.init(&emulator);
    newEmu().init();
    newEmu().on = true;
}


struct Audio {
    int frequency;
}

fn void Audio.init() {

}