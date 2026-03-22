CREATE TABLE quizzes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
    title       TEXT NOT NULL
);

CREATE TABLE quiz_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
    quiz_id     UUID REFERENCES quizzes(id),
    started_at  TIMESTAMPTZ DEFAULT NOW(),
    ended_at    TIMESTAMPTZ
);

CREATE TABLE answer_types (
    id    serial PRIMARY KEY NOT NULL,
    text  TEXT NOT NULL,
    title TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_answer_types_text ON answer_types (lower(text));

INSERT INTO answer_types (text, title)
VALUES ('Reading', 'What is the reading of this word?'), ('Meaning', 'What is the meaning of this word?');

CREATE TABLE questions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
    quiz_id           UUID REFERENCES quizzes(id),
    type_id     		int4 REFERENCES answer_types(id) NOT NULL,
    kanji             TEXT NOT NULL,
    correct_answer_id UUID  -- FK to answers, set after insert
);

CREATE TABLE answers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
    question_id UUID REFERENCES questions(id) NOT NULL,
    text        TEXT NOT NULL
);

CREATE TABLE users (
    id        	UUID PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
    name 		TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_users_name ON users (lower(name));

CREATE TABLE participants (
    id        	    UUID PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
    user_id  		UUID REFERENCES users(id) NOT NULL, -- create new user if name wasn't found
    session_id  	UUID REFERENCES quiz_sessions(id) NOT NULL
);

CREATE UNIQUE INDEX idx_participants_user_id ON participants (user_id, session_id);

CREATE TABLE submissions (
    taken_ms  INT DEFAULT id             UUID PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
    participant_id UUID REFERENCES participants(id) NOT NULL,
    question_id    UUID REFERENCES questions(id) NOT NULL,
    answer_id      UUID REFERENCES answers(id),
    is_correct     BOOLEAN NOT NULL DEFAULT FALSE,
    time_0 CHECK (time_taken_ms >= 0),
    score          INT NOT NULL DEFAULT 0 CHECK (score >= 0)
);

ALTER TABLE submissions ADD CONSTRAINT uniq_participant_question UNIQUE (participant_id, question_id);
