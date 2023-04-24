CREATE TABLE IF NOT EXISTS entries(
   id SERIAL PRIMARY KEY,
   text VARCHAR,
   creator_id VARCHAR (50) NOT NULL,
   created_at TIMESTAMP NOT NULL,
   updated_at TIMESTAMP,
   embedding vector(1536)
);