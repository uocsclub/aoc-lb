
create table modifiers (
    language_name TEXT PRIMARY KEY NOT NULL,
    modifier_dec_percent INTEGER 
);

create table modifier_submission (
    year VARCHAR(5), -- I hope this breaks in 7975 years
    user_id INTEGER NOT NULL REFERENCES aoc_user(id),
    day TEXT NOT NULL, -- 03d2 format 2nd star of 3rd day
    submission_url TEXT NOT NULL,
    language_name TEXT NOT NULL REFERENCES modifiers(language_name)
);

INSERT INTO modifiers (language_name, modifier_dec_percent) VALUES 
    ("Custom made language", 50),
    ("VHDL", 40),
    ("Verilog", 40),
    ("SystemVerilog", 40),
    ("Haskell", 30),
    ("Elixir", 30),
    ("APL", 30),
    ("J", 30),
    ("K", 30),
    ("Q", 30),
    ("uiua", 30),
    ("Prolog", 30),
    ("Lean", 30),
    ("F#", 25),
    ("OCaml", 25),
    ("Matlab", 25),
    ("R", 25),
    ("Ada", 20),
    ("Cobol", 20),
    ("SQL", 15),
    ("Powershell", 15),
    ("Bash", 15),
    ("Fish", 15),
    ("Batch", 15),
    ("Commonlisp", 20),
    ("Scheme", 20),
    ("C", 10),

-- normie loser languages
    ("Python", 0),
    ("Java", 0),
    ("C#", 0),
    ("C++", 0),
    ("Rust", 0),
    ("Javascript", 0),
    ("Typescript", 0),
    ("Lua", 0),
    ("Zig", 0),
    ("Visual basic", 0),
    ("Swift", 0),
    ("Gleam", 0),
    ("Go", 0),
    ("Dart", 0),
    ("Odin", 0),
    ("JAI", 0),
    ("PHP", 0),
    ("Ruby", 0),
    ("Kotlin", 0);
