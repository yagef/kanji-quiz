CREATE TABLE quizzes (
                             id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                             title     TEXT NOT NULL
);

CREATE TABLE questions (
                            id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                            quiz_id          UUID REFERENCES quizzes(id),
                            kanji            TEXT NOT NULL,
                            order_index      INT NOT NULL,
                            correct_answer_id UUID  -- FK to answers, set after insert
);

CREATE TABLE answers (
                             id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                             question_id UUID REFERENCES questions(id),
                             text        TEXT NOT NULL  -- reading or meaning
);

CREATE TABLE decoy_words (
                             id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                             quiz_id UUID REFERENCES quizzes(id),
                             text    TEXT NOT NULL
);

CREATE TABLE participants (
                              id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                              quiz_id  UUID REFERENCES quizzes(id),
                              username TEXT NOT NULL,
                              token    TEXT UNIQUE NOT NULL,  -- session token from QR scan
                              score    INT DEFAULT 0
);

CREATE TABLE submissions (
                             id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                             participant_id   UUID REFERENCES participants(id),
                             question_id      UUID REFERENCES questions(id),
                             chosen_answer_id UUID,   -- NULL = timed out
                             is_correct       BOOLEAN,
                             time_taken_ms    INT
);

-- Tracks which answer combo each participant saw (for deduplication)
CREATE TABLE answer_assignments (
                            participant_id UUID REFERENCES participants(id),
                            question_id    UUID REFERENCES questions(id),
                            answer_options UUID[],  -- array of 4 answer IDs shown
                            PRIMARY KEY (participant_id, question_id)
);