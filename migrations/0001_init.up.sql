CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       telegram_id BIGINT UNIQUE NOT NULL,
                       username TEXT,
                       created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE meetings (
                          id SERIAL PRIMARY KEY,
                          user_id INT REFERENCES users(id) ON DELETE CASCADE,
                          file_name TEXT NOT NULL,
                          transcription TEXT,
                          summary TEXT,
                          created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX idx_meetings_transcription ON meetings USING gin(to_tsvector('russian', transcription));