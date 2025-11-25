
CREATE TABLE aoc_user (
    aoc_id INTEGER PRIMARY KEY NOT NULL, -- aoc is probably consistent with their IDs
    name TEXT,
    github_id INTEGER DEFAULT NULL,
    avatar_url TEXT NOT NULL DEFAULT ''
);

CREATE TABLE leaderboard_entry (
    year VARCHAR(5), -- I hope this breaks in 7975 years
    user_id INTEGER NOT NULL REFERENCES aoc_user(id),
    score INTEGER NOT NULL,
    day_completions TEXT, -- comma separated in the format of 01d1 for 1st star of 1st day

    PRIMARY KEY(year, user_id)
);

-- For custom user submissions with custom languages
CREATE TABLE submission (
    id INTEGER PRIMARY KEY NOT NULL,
    leaderboard_entry INTEGER NOT NULL REFERENCES leaderboard_entry(id),
    language TEXT NOT NULL,
    url TEXT
);
